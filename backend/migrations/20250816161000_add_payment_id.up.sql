-- Add payment_id column to booking_payments table
ALTER TABLE booking_payments ADD COLUMN payment_id text not null;

-- Create index for payment_id
CREATE INDEX idx_booking_payments_payment ON booking_payments(payment_id) WHERE payment_id IS NOT NULL;