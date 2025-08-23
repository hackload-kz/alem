package portriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"

	"hackload/internal/sqlc"
	"hackload/pkg/eventprovider"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

type ConfirmOrderArgs struct {
	BookingID      int64
	OrderID        uuid.UUID
	ExpectedPlaces int
}

func (ConfirmOrderArgs) Kind() string { return "booking.confirm_order" }

type ConfirmOrderWorker struct {
	river.WorkerDefaults[ConfirmOrderArgs]

	queries       *sqlc.Queries
	db            *sql.DB
	EventProvider eventprovider.ClientInterface
}

func NewConfirmOrderWorker(queries *sqlc.Queries, db *sql.DB, eventProvider eventprovider.ClientInterface) river.Worker[ConfirmOrderArgs] {
	return &ConfirmOrderWorker{
		queries:       queries,
		db:            db,
		EventProvider: eventProvider,
	}
}

func (w *ConfirmOrderWorker) Work(ctx context.Context, job *river.Job[ConfirmOrderArgs]) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := w.queries.WithTx(tx)

	// 1. Get the booking to verify it exists
	booking, err := qtx.GetBooking(ctx, job.Args.BookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking: %w", err)
	}

	// 2. Get order details and validate expected number of places
	getOrderResp, err := w.EventProvider.GetOrder(ctx, job.Args.OrderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	var order eventprovider.Order
	if err := json.NewDecoder(getOrderResp.Body).Decode(&order); err != nil {
		return fmt.Errorf("failed to decode getOrder response: %w", err)
	}

	fmt.Printf("getOrder: %#v\n", order)

	// Validate that the order has the expected number of selected places
	if order.PlacesCount != job.Args.ExpectedPlaces {
		return fmt.Errorf("order validation failed: expected %d places, got %d", job.Args.ExpectedPlaces, order.PlacesCount)
	}

	if slices.Contains([]eventprovider.OrderStatus{
		eventprovider.CANCELLED,
		eventprovider.CONFIRMED,
	}, order.Status) {
		return nil
	}

	// 3. Submit order
	if order.Status == eventprovider.STARTED {
		submitResp, err := w.EventProvider.SubmitOrder(ctx, job.Args.OrderID)
		if err != nil {
			return fmt.Errorf("failed to submit order: %w", err)
		}

		fmt.Printf("submitResp: %v\n", submitResp.StatusCode)

		if submitResp.StatusCode > 299 {
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
	}

	// 4. Confirm order
	confirmResp, err := w.EventProvider.ConfirmOrder(ctx, job.Args.OrderID)
	if err != nil {
		return fmt.Errorf("failed to confirm order: %w", err)
	}

	if confirmResp.StatusCode > 299 {
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

	// 5. Get all seats for this booking and update them to SOLD
	seatIDs, err := qtx.GetBookingSeats(ctx, booking.ID)
	if err != nil {
		return fmt.Errorf("failed to get booking seats: %w", err)
	}

	if len(seatIDs) > 0 {
		err = qtx.UpdateSeatsStatusByIDs(ctx, sqlc.UpdateSeatsStatusByIDsParams{
			Status:  "SOLD",
			SeatIds: seatIDs,
		})
		if err != nil {
			return fmt.Errorf("failed to update seats status: %w", err)
		}
	}

	return tx.Commit()
}
