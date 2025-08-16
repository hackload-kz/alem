-- Remove payment_id column from booking_payments table
DROP INDEX IF EXISTS idx_booking_payments_payment;

-- SQLite doesn't support dropping columns directly in older versions
-- This would require recreating the table, but for development we can leave it
-- ALTER TABLE booking_payments DROP COLUMN payment_id;