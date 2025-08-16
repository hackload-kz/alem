package portriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"hackload/internal/config"
	"hackload/internal/paymenttoken"
	"hackload/internal/sqlc"
	"hackload/pkg/paymentgateway"

	"github.com/riverqueue/river"
)

type RefundPaymentArgs struct {
	BookingID int64
}

func (RefundPaymentArgs) Kind() string { return "payment.refund" }

type RefundPaymentWorker struct {
	river.WorkerDefaults[RefundPaymentArgs]

	queries        *sqlc.Queries
	db             *sql.DB
	paymentGateway paymentgateway.ClientInterface
	config         *config.Config
}

func NewRefundPaymentWorker(queries *sqlc.Queries, db *sql.DB, paymentGateway paymentgateway.ClientInterface, config *config.Config) river.Worker[RefundPaymentArgs] {
	return &RefundPaymentWorker{
		queries:        queries,
		db:             db,
		paymentGateway: paymentGateway,
		config:         config,
	}
}

func (w *RefundPaymentWorker) Work(ctx context.Context, job *river.Job[RefundPaymentArgs]) error {
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

	// 2. Get payment record for this booking
	payment, err := qtx.GetBookingPaymentByBookingID(ctx, booking.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No payment record found, nothing to refund
			return tx.Commit()
		}
		return fmt.Errorf("failed to get booking payment: %w", err)
	}

	// 3. Check if payment was successful and can be refunded
	if payment.Status == nil || *payment.Status != "SUCCESS" {
		// Payment wasn't successful, nothing to refund
		return tx.Commit()
	}

	// Generate token for cancel operation (same parameters as init)
	token := paymenttoken.GenerateToken(
		payment.Amount,
		payment.Currency,
		payment.OrderID, // Use the original OrderID
		w.config.PaymentProvider.MerchantPassword,
		payment.TeamSlug,
	)

	// 4. Call payment gateway to cancel/refund the payment
	cancelResp, err := w.paymentGateway.PostApiV1PaymentCancelCancel(ctx, paymentgateway.PaymentCancelRequestDto{
		PaymentId: payment.PaymentID, // Use the actual PaymentID from gateway
		TeamSlug:  payment.TeamSlug,  // Use the saved TeamSlug
		Token:     token,             // Generate token using saved parameters
	})
	if err != nil {
		return fmt.Errorf("failed to cancel payment: %w", err)
	}

	if cancelResp.StatusCode != 200 {
		return fmt.Errorf("failed to cancel payment, status: %d", cancelResp.StatusCode)
	}

	// Parse cancel response to check if it was successful
	var cancelResult map[string]any
	if err := json.NewDecoder(cancelResp.Body).Decode(&cancelResult); err != nil {
		return fmt.Errorf("failed to decode cancel response: %w", err)
	}

	// 5. Update payment status to REFUNDED
	refundedStatus := "REFUNDED"
	err = qtx.UpdateBookingPaymentStatus(ctx, sqlc.UpdateBookingPaymentStatusParams{
		Status:  &refundedStatus,
		OrderID: payment.OrderID,
	})
	if err != nil {
		return fmt.Errorf("failed to update payment status to REFUNDED: %w", err)
	}

	return tx.Commit()
}
