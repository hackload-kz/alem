-- Add columns needed for token generation
ALTER TABLE booking_payments ADD COLUMN amount integer not null; -- Amount in cents
ALTER TABLE booking_payments ADD COLUMN currency text not null; -- Currency (e.g., KZT)
ALTER TABLE booking_payments ADD COLUMN team_slug text not null; -- TeamSlug/MerchantID