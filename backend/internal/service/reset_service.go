package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

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
	allPlaces, err := s.fetchPlacesParallel(ctx)
	if err != nil {
		slog.Error("unable to get places", "error", err)
		return err
	}

	slog.Info("starting preloader process")

	// Start database transaction
	tx, err := s.db.Begin()
	if err != nil {
		slog.Error("unable to start transaction", "error", err)
		return err
	}
	defer tx.Rollback()

	txQueries := s.queries.WithTx(tx)

	// 1. Remove ALL seats and related booking data
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

	// 2. Get all places from EventProvider (already done above)
	slog.Info("fetched places from EventProvider", "count", len(allPlaces))

	// 3. Insert places as seats into database
	slog.Info("inserting places as seats")

	// Insert in chunks to avoid SQL parameter limit
	const chunkSize = 1000 // SQLite can handle ~32768 parameters, with 6 columns = ~5000 rows max
	totalChunks := (len(allPlaces) + chunkSize - 1) / chunkSize

	for chunkIndex := range totalChunks {
		start := chunkIndex * chunkSize
		end := min(start+chunkSize, len(allPlaces))

		chunk := allPlaces[start:end]
		slog.Info("inserting chunk", "chunk", chunkIndex+1, "total_chunks", totalChunks, "size", len(chunk))

		// Build batch insert query for this chunk
		insertQuery := sq.Insert("seats").Columns("event_id", "external_id", "row", "number", "price", "status")

		for _, place := range chunk {
			status := "FREE"
			if !place.IsFree {
				status = "RESERVED"
			}

			// Calculate price based on seat position (row * 1000 + seat number)
			price := calculateSeatPrice(place.Row, place.Seat)

			// Note: We need an event_id, but since this is a preloader and no specific event is mentioned,
			// we'll use event_id = 1. In a real scenario, this should be parameterized.
			externalID := place.Id.String()
			insertQuery = insertQuery.Values(1, externalID, int64(place.Row), int64(place.Seat), price, status)
		}

		// Execute batch insert for this chunk
		sql, args, err := insertQuery.ToSql()
		if err != nil {
			slog.Error("unable to build batch insert query", "chunk", chunkIndex+1, "error", err)
			return err
		}

		_, err = tx.ExecContext(ctx, sql, args...)
		if err != nil {
			slog.Error("unable to execute batch insert", "chunk", chunkIndex+1, "error", err)
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		slog.Error("unable to commit transaction", "error", err)
		return err
	}

	slog.Info("preloader process completed successfully", "seats_inserted", len(allPlaces))
	return nil
}

func calculateSeatPrice(row, seat int) string {
	// Calculate seat index based on row and seat number (each row has 1000 seats)
	seatIndex := row*1000 + seat
	switch {
	case seatIndex <= 10000: // 0-9,999: Золотой круг
		return "40000.00"
	case seatIndex <= 25000: // 10,000-24,999: Фан-зона
		return "80000.00"
	case seatIndex <= 45000: // 25,000-44,999: Нижний ярус
		return "120000.00"
	case seatIndex <= 70000: // 45,000-69,999: Средний ярус
		return "160000.00"
	default: // 70,000+: Верхний ярус
		return "200000.00"
	}
}

// fetchPlacesParallel fetches exactly 100 pages using 6 goroutines in parallel
func (s *resetService) fetchPlacesParallel(ctx context.Context) ([]eventprovider.Place, error) {
	var (
		numWorkers = 5
		totalPages = 100
		pageSize   = 1000
	)

	// Calculate pages per worker
	pagesPerWorker := totalPages / numWorkers
	remainingPages := totalPages % numWorkers

	var wg sync.WaitGroup
	var mu sync.Mutex
	var allPlaces []eventprovider.Place
	var globalErr error

	for workerID := 0; workerID < numWorkers; workerID++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			// Calculate page range for this worker
			// First 'remainingPages' workers get an extra page
			var startPage, endPage int

			if id < remainingPages {
				// Workers 0-3 get 17 pages each (16 base + 1 extra)
				startPage = id*(pagesPerWorker+1) + 1
				endPage = startPage + pagesPerWorker
			} else {
				// Workers 4-5 get 16 pages each
				startPage = remainingPages*(pagesPerWorker+1) + (id-remainingPages)*pagesPerWorker + 1
				endPage = startPage + pagesPerWorker - 1
			}

			// Worker distribution:
			// Worker 0: pages 1-17   (17 pages)
			// Worker 1: pages 18-34  (17 pages)
			// Worker 2: pages 35-51  (17 pages)
			// Worker 3: pages 52-68  (17 pages)
			// Worker 4: pages 69-84  (16 pages)
			// Worker 5: pages 85-100 (16 pages)

			workerPlaces := []eventprovider.Place{}

			for page := startPage; page <= endPage; page++ {
				// Check if another worker encountered an error
				mu.Lock()
				if globalErr != nil {
					mu.Unlock()
					return
				}
				mu.Unlock()

				slog.Info("worker fetching page", "worker", id, "page", page)

				placesResp, err := s.eventProviderWithResponses.ListPlacesWithResponse(ctx, &eventprovider.ListPlacesParams{
					Page:     &page,
					PageSize: &pageSize,
				})
				if err != nil {
					slog.Error("unable to get places", "worker", id, "error", err, "page", page)
					mu.Lock()
					if globalErr == nil {
						globalErr = fmt.Errorf("worker %d failed on page %d: %w", id, page, err)
					}
					mu.Unlock()
					return
				}

				if placesResp.StatusCode() != 200 {
					slog.Error("bad response from event provider", "worker", id, "status", placesResp.StatusCode(), "page", page)
					mu.Lock()
					if globalErr == nil {
						globalErr = fmt.Errorf("worker %d got status %d on page %d", id, placesResp.StatusCode(), page)
					}
					mu.Unlock()
					return
				}

				if placesResp.JSON200 == nil {
					slog.Error("no places data in response", "worker", id, "page", page)
					mu.Lock()
					if globalErr == nil {
						globalErr = fmt.Errorf("worker %d got no data on page %d", id, page)
					}
					mu.Unlock()
					return
				}

				places := *placesResp.JSON200
				workerPlaces = append(workerPlaces, places...)

				slog.Info("worker fetched places page", "worker", id, "page", page, "count", len(places))
			}

			// Add this worker's places to the global slice
			mu.Lock()
			allPlaces = append(allPlaces, workerPlaces...)
			mu.Unlock()

			slog.Info("worker completed", "worker", id, "pages", fmt.Sprintf("%d-%d", startPage, endPage), "places", len(workerPlaces))
		}(workerID)
	}

	// Wait for all workers to complete
	wg.Wait()

	if globalErr != nil {
		return nil, globalErr
	}

	slog.Info("fetched all places", "total", len(allPlaces))
	return allPlaces, nil
}
