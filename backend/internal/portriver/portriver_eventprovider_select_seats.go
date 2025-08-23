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

type SelectSeatsArgs struct {
	BookingID int64
}

func (SelectSeatsArgs) Kind() string { return "booking.select_seats" }

type SelectSeatsWorker struct {
	river.WorkerDefaults[SelectSeatsArgs]

	queries       *sqlc.Queries
	db            *sql.DB
	riverClient   *river.Client[*sql.Tx]
	EventProvider eventprovider.ClientInterface
}

func NewSelectSeatsWorker(queries *sqlc.Queries, db *sql.DB, riverClient *river.Client[*sql.Tx], eventProvider eventprovider.ClientInterface) river.Worker[SelectSeatsArgs] {
	return &SelectSeatsWorker{
		queries:       queries,
		db:            db,
		riverClient:   riverClient,
		EventProvider: eventProvider,
	}
}

func (w *SelectSeatsWorker) Work(ctx context.Context, job *river.Job[SelectSeatsArgs]) error {
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
	fmt.Printf("orderCreated: %v\n", orderCreated)

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
	selectedSeats := 0
	for _, seatID := range seatIDs {
		// Get seat details efficiently by ID
		targetSeat, err := qtx.GetSeatByID(ctx, seatID)
		if err != nil {
			return fmt.Errorf("failed to get seat %d: %w", seatID, err)
		}

		fmt.Printf("targetSeat: %#v\n", targetSeat)

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

		fmt.Printf("selectResp: %v\n", selectResp.StatusCode)

		if selectResp.StatusCode > 299 {
			return fmt.Errorf("failed to select place, status: %d", selectResp.StatusCode)
		}

		selectedSeats++
	}

	// Commit the transaction first to ensure seat selections are persisted
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seat selection transaction: %w", err)
	}

	// 6. Enqueue ConfirmOrder worker to handle order validation, submission and confirmation
	_, err = w.riverClient.Insert(ctx, ConfirmOrderArgs{
		BookingID:      booking.ID,
		OrderID:        orderID,
		ExpectedPlaces: selectedSeats,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to enqueue confirm order job: %w", err)
	}

	return nil
}
