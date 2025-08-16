package portriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"hackload/internal/sqlc"
	"hackload/pkg/eventprovider"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

type ConfirmBookingArgs struct {
	BookingID int64
}

func (ConfirmBookingArgs) Kind() string { return "booking.confirm" }

type ConfirmBookingWorker struct {
	river.WorkerDefaults[ConfirmBookingArgs]

	queries       *sqlc.Queries
	db            *sql.DB
	EventProvider eventprovider.ClientInterface
}

func NewConfirmBookingWorker(queries *sqlc.Queries, db *sql.DB, eventProvider eventprovider.ClientInterface) river.Worker[ConfirmBookingArgs] {
	return &ConfirmBookingWorker{
		queries:       queries,
		db:            db,
		EventProvider: eventProvider,
	}
}

func (w *ConfirmBookingWorker) Work(ctx context.Context, job *river.Job[ConfirmBookingArgs]) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := w.queries.WithTx(tx)

	// 1. Get the booking
	booking, err := qtx.GetBooking(ctx, job.Args.BookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking: %w", err)
	}

	// 2. Get all seats for this booking
	seatIDs, err := qtx.GetBookingSeats(ctx, booking.ID)
	if err != nil {
		return fmt.Errorf("failed to get booking seats: %w", err)
	}

	if len(seatIDs) == 0 {
		return fmt.Errorf("no seats found for booking %d", booking.ID)
	}

	// 3. Start order with EventProvider
	orderResp, err := w.EventProvider.StartOrder(ctx)
	if err != nil {
		return fmt.Errorf("failed to start order: %w", err)
	}

	if orderResp.StatusCode != 201 {
		return fmt.Errorf("failed to start order, status: %d", orderResp.StatusCode)
	}

	// Parse order creation response
	var orderCreated eventprovider.OrderCreatedResponse
	if err := json.NewDecoder(orderResp.Body).Decode(&orderCreated); err != nil {
		return fmt.Errorf("failed to decode order response: %w", err)
	}

	orderID := orderCreated.OrderId

	// 4. Save order_id into booking_orders
	err = qtx.InsertBookingOrder(ctx, sqlc.InsertBookingOrderParams{
		BookingID: booking.ID,
		OrderID:   orderID.String(),
		Status:    stringPtr("STARTED"),
	})
	if err != nil {
		return fmt.Errorf("failed to insert booking order: %w", err)
	}

	// 5. For each seat, get seat info and select it
	for _, seatID := range seatIDs {
		// Get seat details efficiently by ID
		targetSeat, err := qtx.GetSeatByID(ctx, seatID)
		if err != nil {
			return fmt.Errorf("failed to get seat %d: %w", seatID, err)
		}

		// Skip seats without external_id
		if targetSeat.ExternalID == nil {
			continue
		}

		placeID, err := uuid.Parse(*targetSeat.ExternalID)
		if err != nil {
			return fmt.Errorf("failed to parse external_id: %w", err)
		}

		// Select place for the order
		selectResp, err := w.EventProvider.SelectPlace(ctx, placeID, eventprovider.SelectPlaceRequest{
			OrderId: orderID,
		})
		if err != nil {
			return fmt.Errorf("failed to select place: %w", err)
		}

		if selectResp.StatusCode != 200 {
			return fmt.Errorf("failed to select place, status: %d", selectResp.StatusCode)
		}
	}

	// 6. Submit order
	submitResp, err := w.EventProvider.SubmitOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to submit order: %w", err)
	}

	if submitResp.StatusCode != 200 {
		return fmt.Errorf("failed to submit order, status: %d", submitResp.StatusCode)
	}

	// Update booking order status to SUBMITTED
	err = qtx.UpdateBookingOrderStatus(ctx, sqlc.UpdateBookingOrderStatusParams{
		Status:    stringPtr("SUBMITTED"),
		BookingID: booking.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update booking order status to SUBMITTED: %w", err)
	}

	// 7. Confirm order
	confirmResp, err := w.EventProvider.ConfirmOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to confirm order: %w", err)
	}

	if confirmResp.StatusCode != 200 {
		return fmt.Errorf("failed to confirm order, status: %d", confirmResp.StatusCode)
	}

	// Update booking order status to CONFIRMED
	err = qtx.UpdateBookingOrderStatus(ctx, sqlc.UpdateBookingOrderStatusParams{
		Status:    stringPtr("CONFIRMED"),
		BookingID: booking.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update booking order status to CONFIRMED: %w", err)
	}

	// 9. Update all seats status to SOLD
	err = qtx.UpdateSeatsStatusByIDs(ctx, sqlc.UpdateSeatsStatusByIDsParams{
		Status:  "SOLD",
		SeatIds: seatIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to update seats status: %w", err)
	}

	return tx.Commit()
}
