package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	SQLite3Path string `env:"SQLITE3_PATH"`

	// API
	API struct {
		Port string `env:"PORT, default=8080"`
		Addr string `env:"ADDR"`
	} `env:", prefix=API_"`

	// Провайдер билетов (Event Provider)
	EventProvider struct {
		Addr string `env:"ADDR"`
	} `env:", prefix=EVENT_PROVIDER_"`

	// API Платежного шлюза
	PaymentProvider struct {
		Addr             string `env:"ADDR"`
		MerchantPassword string `env:"MERCHANT_PASSWORD"`
		MerchantID       string `env:"MERCHANT_ID"`
	} `env:", prefix=PAYMENT_PROVIDER_"`
}

func GetConfig(ctx context.Context) (*Config, error) {
	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
