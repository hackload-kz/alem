package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"hackload/internal/sqlc"
)

type AuthenticationService interface {
	GetSession(ctx context.Context, bearerToken string) (*GetSessionResponse, error)
}

type GetSessionResponse struct {
	UserID int64
	Email  string
}

var ErrUnauthorized = fmt.Errorf("unauthorized")

type authenticationService struct {
	queries *sqlc.Queries
}

func NewAuthenticationService(queries *sqlc.Queries) AuthenticationService {
	return &authenticationService{
		queries: queries,
	}
}

func (s *authenticationService) GetSession(ctx context.Context, bearerToken string) (*GetSessionResponse, error) {
	decoded, err := base64.StdEncoding.DecodeString(bearerToken)
	if err != nil {
		return nil, ErrUnauthorized
	}

	credentials := string(decoded)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		return nil, ErrUnauthorized
	}

	email := parts[0]
	password := parts[1]

	user, err := s.queries.GetUser(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUnauthorized
	}

	if user.PasswordPlain != nil && *user.PasswordPlain == password {
		return &GetSessionResponse{
			UserID: user.UserID,
			Email:  user.Email,
		}, nil
	}

	hasher := sha256.New()
	hasher.Write([]byte(password))
	hashedPassword := hex.EncodeToString(hasher.Sum(nil))

	if user.PasswordHash != hashedPassword {
		return nil, ErrUnauthorized
	}

	return &GetSessionResponse{
		UserID: user.UserID,
		Email:  user.Email,
	}, nil
}
