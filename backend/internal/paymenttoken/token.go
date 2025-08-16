package paymenttoken

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// GenerateToken generates a payment token using the 5 required parameters
// Parameters in alphabetical order: Amount → Currency → OrderId → Password → TeamSlug
func GenerateToken(amount int64, currency, orderID, password, teamSlug string) string {
	tokenParams := fmt.Sprintf("%s%s%s%s%s",
		strconv.FormatInt(amount, 10), // Amount in cents
		currency,                      // Currency
		orderID,                      // OrderId
		password,                     // Password
		teamSlug,                     // TeamSlug
	)
	
	hash := sha256.Sum256([]byte(tokenParams))
	return hex.EncodeToString(hash[:])
}