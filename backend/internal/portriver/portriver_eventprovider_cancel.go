package portriver

import (
	"context"
	"database/sql"
	"fmt"

	"hackload/internal/sqlc"
	"hackload/pkg/eventprovider"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

type CancelBookingArgs struct {
	BookingID int64
}

func (CancelBookingArgs) Kind() string { return "booking.cancel" }

type CancelBookingWorker struct {
	river.WorkerDefaults[CancelBookingArgs]

	queries       *sqlc.Queries
	db            *sql.DB
	riverClient   *river.Client[*sql.Tx]
	EventProvider eventprovider.ClientInterface
}

func NewCancelBookingWorker(queries *sqlc.Queries, db *sql.DB, riverClient *river.Client[*sql.Tx], eventProvider eventprovider.ClientInterface) river.Worker[CancelBookingArgs] {
	return &CancelBookingWorker{
		queries:       queries,
		db:            db,
		riverClient:   riverClient,
		EventProvider: eventProvider,
	}
}

func (w *CancelBookingWorker) Work(ctx context.Context, job *river.Job[CancelBookingArgs]) error {
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
		// No seats to cancel, just return success
		return tx.Commit()
	}

	// 3. Get existing booking order if it exists
	var orderID *uuid.UUID
	bookingOrder, err := qtx.GetBookingOrder(ctx, booking.ID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get booking order: %w", err)
	}
	if err == nil {
		parsed, err := uuid.Parse(bookingOrder.OrderID)
		if err != nil {
			return fmt.Errorf("failed to parse order_id: %w", err)
		}
		orderID = &parsed
	}

	// 4. For each seat, release it from EventProvider
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

		// Release place in EventProvider
		releaseResp, err := w.EventProvider.ReleasePlace(ctx, placeID)
		if err != nil {
			return fmt.Errorf("failed to release place: %w", err)
		}

		if releaseResp.StatusCode != 200 {
			return fmt.Errorf("failed to release place, status: %d", releaseResp.StatusCode)
		}
	}

	// 5. Cancel order in EventProvider if it exists
	if orderID != nil {
		cancelResp, err := w.EventProvider.CancelOrder(ctx, *orderID)
		if err != nil {
			return fmt.Errorf("failed to cancel order: %w", err)
		}

		if cancelResp.StatusCode != 200 {
			return fmt.Errorf("failed to cancel order, status: %d", cancelResp.StatusCode)
		}

		// Update booking order status to CANCELLED
		err = qtx.UpdateBookingOrderStatus(ctx, sqlc.UpdateBookingOrderStatusParams{
			Status:    stringPtr("CANCELLED"),
			BookingID: booking.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to update booking order status to CANCELLED: %w", err)
		}
	}

	// Commit the transaction first
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 6. Queue ReleaseSeatsWorker to update seat statuses to FREE
	_, err = w.riverClient.Insert(ctx, ReleaseSeatsArgs{
		BookingID: booking.ID,
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to queue ReleaseSeatsWorker: %w", err)
	}

	return nil
}

func stringPtr(s string) *string {
	return &s
}
