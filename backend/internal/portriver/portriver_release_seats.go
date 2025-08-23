package portriver

import (
	"context"
	"database/sql"
	"fmt"

	"hackload/internal/sqlc"

	"github.com/riverqueue/river"
)

type ReleaseSeatsArgs struct {
	BookingID int64
	StatusEq  *string
}

func (ReleaseSeatsArgs) Kind() string { return "booking.release_seats" }

type ReleaseSeatsWorker struct {
	river.WorkerDefaults[ReleaseSeatsArgs]

	queries *sqlc.Queries
	db      *sql.DB
}

func NewReleaseSeatsWorker(queries *sqlc.Queries, db *sql.DB) river.Worker[ReleaseSeatsArgs] {
	return &ReleaseSeatsWorker{
		queries: queries,
		db:      db,
	}
}

func (w *ReleaseSeatsWorker) Work(ctx context.Context, job *river.Job[ReleaseSeatsArgs]) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for booking %d: %w", job.Args.BookingID, err)
	}
	defer tx.Rollback()

	qtx := w.queries.WithTx(tx)

	// 1. GetBooking to verify it exists
	booking, err := qtx.GetBooking(ctx, job.Args.BookingID)
	if err != nil {
		return fmt.Errorf("failed to get booking %d: %w", job.Args.BookingID, err)
	}

	if job.Args.StatusEq != nil && *job.Args.StatusEq != booking.Status {
		return nil
	}

	// 2. Get all seat IDs for this booking
	seatIDs, err := qtx.GetBookingSeats(ctx, booking.ID)
	if err != nil {
		return fmt.Errorf("failed to get booking seats for booking %d: %w", booking.ID, err)
	}

	// 3. Delete booking_seats records
	rowsAffected, err := qtx.DeleteBookingSeats(ctx, booking.ID)
	if err != nil {
		return fmt.Errorf("failed to delete booking seats for booking %d: %w", booking.ID, err)
	}

	// 4. Update seats status to FREE (only if we actually deleted booking seats)
	if rowsAffected > 0 && len(seatIDs) > 0 {
		err = qtx.UpdateSeatsStatusByIDs(ctx, sqlc.UpdateSeatsStatusByIDsParams{
			Status:  "FREE",
			SeatIds: seatIDs,
		})
		if err != nil {
			return fmt.Errorf("failed to update seats status to FREE for booking %d: %w", booking.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for booking %d: %w", booking.ID, err)
	}
	return nil
}
