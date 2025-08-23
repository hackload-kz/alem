package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"hackload/internal/sqlc"
	"hackload/pkg/eventprovider"

	sq "github.com/Masterminds/squirrel"
)

type ResetService interface {
	Reset(ctx context.Context) error
}

type resetService struct {
	queries                    *sqlc.Queries
	db                         *sql.DB
	eventProviderWithResponses *eventprovider.ClientWithResponses
}

func NewResetService(
	queries *sqlc.Queries,
	db *sql.DB,
	eventProviderWithResponses *eventprovider.ClientWithResponses,
) ResetService {
	return &resetService{
		queries:                    queries,
		db:                         db,
		eventProviderWithResponses: eventProviderWithResponses,
	}
}

func (s *resetService) Reset(ctx context.Context) error {
	slog.Info("starting preloader process")

	// Start database transaction
	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("unable to start transaction", "error", err)
		return err
	}
	defer tx.Rollback()

	txQueries := s.queries.WithTx(tx)

	// 1. Clear existing data
	if err := s.clearExistingData(ctx, txQueries); err != nil {
		return err
	}

	// 2. Setup channels for producer-consumer pattern
	type placeChunk struct {
		places []eventprovider.Place
		page   int
	}

	placeChan := make(chan placeChunk, 10) // Buffered channel for place chunks
	errChan := make(chan error, 1)         // Error channel
	doneChan := make(chan struct{})        // Completion signal

	var totalInserted atomic.Int64
	var fetchComplete atomic.Bool

	// Context for cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 3. Start database inserter workers
	const numInserters = 3
	var insertWg sync.WaitGroup

	for i := 0; i < numInserters; i++ {
		insertWg.Add(1)
		go func(workerID int) {
			defer insertWg.Done()

			for chunk := range placeChan {
				if err := s.insertPlaceChunk(ctx, tx, chunk.places, workerID); err != nil {
					slog.Error("insert worker failed", "worker", workerID, "error", err)
					select {
					case errChan <- err:
						cancel() // Cancel context to stop all workers
					default:
					}
					return
				}

				count := int64(len(chunk.places))
				totalInserted.Add(count)
				slog.Info("insert worker processed chunk",
					"worker", workerID,
					"page", chunk.page,
					"places", count,
					"total_inserted", totalInserted.Load())
			}

			slog.Info("insert worker completed", "worker", workerID)
		}(i)
	}

	// 4. Start fetcher workers
	const numFetchers = 5
	const totalPages = 100

	pageChan := make(chan int, totalPages)
	var fetchWg sync.WaitGroup

	// Fill page channel
	go func() {
		for page := 1; page <= totalPages; page++ {
			pageChan <- page
		}
		close(pageChan)
	}()

	// Start fetcher workers
	for i := 0; i < numFetchers; i++ {
		fetchWg.Add(1)
		go func(workerID int) {
			defer fetchWg.Done()

			for page := range pageChan {
				select {
				case <-ctx.Done():
					slog.Info("fetcher worker cancelled", "worker", workerID)
					return
				default:
				}

				places, err := s.fetchPage(ctx, page, workerID)
				if err != nil {
					slog.Error("fetcher worker failed", "worker", workerID, "page", page, "error", err)
					select {
					case errChan <- err:
						cancel() // Cancel context to stop all workers
					default:
					}
					return
				}

				select {
				case placeChan <- placeChunk{places: places, page: page}:
					slog.Info("fetcher worker sent chunk", "worker", workerID, "page", page, "places", len(places))
				case <-ctx.Done():
					return
				}
			}

			slog.Info("fetcher worker completed", "worker", workerID)
		}(i)
	}

	// 5. Monitor completion in separate goroutine
	go func() {
		fetchWg.Wait()
		fetchComplete.Store(true)
		close(placeChan) // Signal inserters that no more data is coming
		slog.Info("all fetchers completed, closing place channel")

		insertWg.Wait()
		slog.Info("all inserters completed")
		close(doneChan)
	}()

	// 6. Wait for completion or error
	select {
	case err := <-errChan:
		cancel() // Ensure all workers stop
		return fmt.Errorf("operation failed: %w", err)
	case <-doneChan:
		// All workers completed successfully
	}

	// 7. Commit transaction
	if err := tx.Commit(); err != nil {
		slog.Error("unable to commit transaction", "error", err)
		return err
	}

	slog.Info("preloader process completed successfully", "seats_inserted", totalInserted.Load())
	return nil
}

func (s *resetService) clearExistingData(ctx context.Context, txQueries *sqlc.Queries) error {
	slog.Info("clearing existing data")

	// Delete in order to respect foreign key constraints
	if _, err := txQueries.DeleteAllBookingOrders(ctx); err != nil {
		slog.Error("unable to delete booking orders", "error", err)
		return err
	}

	if _, err := txQueries.DeleteAllBookingPayments(ctx); err != nil {
		slog.Error("unable to delete booking payments", "error", err)
		return err
	}

	if _, err := txQueries.DeleteAllBookingSeats(ctx); err != nil {
		slog.Error("unable to delete booking seats", "error", err)
		return err
	}

	if _, err := txQueries.DeleteAllBookings(ctx); err != nil {
		slog.Error("unable to delete bookings", "error", err)
		return err
	}

	if _, err := txQueries.DeleteAllSeats(ctx); err != nil {
		slog.Error("unable to delete seats", "error", err)
		return err
	}

	slog.Info("cleared existing data successfully")
	return nil
}

func (s *resetService) fetchPage(ctx context.Context, page int, workerID int) ([]eventprovider.Place, error) {
	pageSize := 1000

	slog.Info("fetching page", "worker", workerID, "page", page)

	placesResp, err := s.eventProviderWithResponses.ListPlacesWithResponse(ctx, &eventprovider.ListPlacesParams{
		Page:     &page,
		PageSize: &pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page %d: %w", page, err)
	}

	if placesResp.StatusCode() != 200 {
		return nil, fmt.Errorf("bad response status %d for page %d", placesResp.StatusCode(), page)
	}

	if placesResp.JSON200 == nil {
		return nil, fmt.Errorf("no data in response for page %d", page)
	}

	places := *placesResp.JSON200
	slog.Info("fetched page successfully", "worker", workerID, "page", page, "count", len(places))

	return places, nil
}

func (s *resetService) insertPlaceChunk(ctx context.Context, tx *sql.Tx, places []eventprovider.Place, workerID int) error {
	if len(places) == 0 {
		return nil
	}

	// Build batch insert query
	insertQuery := sq.Insert("seats").Columns("event_id", "external_id", "row", "number", "price", "status")

	for _, place := range places {
		status := "FREE"
		if !place.IsFree {
			status = "RESERVED"
		}

		price := calculateSeatPrice(place.Row, place.Seat)
		externalID := place.Id.String()

		insertQuery = insertQuery.Values(1, externalID, int64(place.Row), int64(place.Seat), price, status)
	}

	// Execute batch insert
	sql, args, err := insertQuery.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = tx.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}

	return nil
}

func calculateSeatPrice(row, seat int) string {
	// Calculate seat index based on row and seat number (each row has 1000 seats)
	seatIndex := (row-1)*1000 + seat
	switch {
	case seatIndex <= 10000: // 1-10,000: Золотой круг
		return "40000.00"
	case seatIndex <= 25000: // 10,001-25,000: Фан-зона
		return "80000.00"
	case seatIndex <= 45000: // 25,001-45,000: Нижний ярус
		return "120000.00"
	case seatIndex <= 70000: // 45,001-70,000: Средний ярус
		return "160000.00"
	default: // 70,001+: Верхний ярус
		return "200000.00"
	}
}
