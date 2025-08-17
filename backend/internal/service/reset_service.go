package service

import (
	"context"
	"database/sql"
	"log/slog"

	"hackload/internal/sqlc"
	"hackload/pkg/eventprovider"
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
	// Get all places from EventProvider by paginating
	var allPlaces []eventprovider.Place
	page := 1
	pageSize := 1000

	for {
		placesResp, err := s.eventProviderWithResponses.ListPlacesWithResponse(ctx, &eventprovider.ListPlacesParams{
			Page:     &page,
			PageSize: &pageSize,
		})
		if err != nil {
			slog.Error("unable to get places", "error", err, "page", page)
			return err
		}

		if placesResp.StatusCode() != 200 {
			slog.Error("bad response from event provider", "status", placesResp.StatusCode(), "page", page)
			return err
		}

		if placesResp.JSON200 == nil {
			slog.Error("no places data in response", "page", page)
			return err
		}

		places := *placesResp.JSON200
		allPlaces = append(allPlaces, places...)

		slog.Info("fetched places page", "page", page, "count", len(places))

		// Stop if we got less than the page size (last page) or no places
		if len(places) < pageSize || len(places) == 0 {
			break
		}

		page++
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

	for seatIndex, place := range allPlaces {
		status := "FREE"
		if !place.IsFree {
			status = "RESERVED"
		}

		// Calculate price based on seat index
		price := calculateSeatPrice(seatIndex)

		// Note: We need an event_id, but since this is a preloader and no specific event is mentioned,
		// we'll use event_id = 1. In a real scenario, this should be parameterized.
		externalID := place.Id.String()
		err := txQueries.InsertSeat(ctx, sqlc.InsertSeatParams{
			EventID:    1, // Default event ID - should be parameterized in production
			ExternalID: &externalID,
			Row:        int64(place.Row),
			Number:     int64(place.Seat),
			Price:      price,
			Status:     status,
		})
		if err != nil {
			slog.Error("unable to insert seat", "place_id", place.Id, "error", err)
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

func calculateSeatPrice(seatIndex int) string {
	switch {
	case seatIndex < 10000: // 0-9,999: Золотой круг
		return "40000.00"
	case seatIndex < 25000: // 10,000-24,999: Фан-зона
		return "80000.00"
	case seatIndex < 45000: // 25,000-44,999: Нижний ярус
		return "120000.00"
	case seatIndex < 70000: // 45,000-69,999: Средний ярус
		return "160000.00"
	default: // 70,000+: Верхний ярус
		return "200000.00"
	}
}
