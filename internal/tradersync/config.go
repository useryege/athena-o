package tradersync

import (
	"fmt"
	"github.com/useryege/athena/internal/accountstate/schema"
	"os"
	"strconv"
	"time"
)

// Config belongs to the independent Trader Sync process.
type Config struct {
	AccountStateDSN                 string
	ShutdownTimeout                 time.Duration
	WebSocketURL, ProxyURL          string
	HTTPURL, SiteURL, CursorHMACKey string
	Projector                       ProjectorConfig
	ReconnectMin, ReconnectMax      time.Duration
	OnError                         func(error)
}

func LoadConfigFromEnv() (Config, error) {
	dsn, err := schema.LoadDSN(os.LookupEnv)
	if err != nil {
		return Config{}, err
	}
	c := Config{AccountStateDSN: dsn, ShutdownTimeout: 30 * time.Second, HTTPURL: os.Getenv("ATHENA_TRADER_SYNC_HTTP_URL"), WebSocketURL: os.Getenv("ATHENA_TRADER_SYNC_WSS_URL"), ProxyURL: os.Getenv("ATHENA_TRADER_SYNC_PROXY_URL"), SiteURL: os.Getenv("ATHENA_URL"), CursorHMACKey: os.Getenv("ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY"), Projector: ProjectorConfig{Interval: 2 * time.Second, MetadataWait: 2 * time.Second, MetadataTimeout: 30 * time.Second, MaxInFlightSources: 100}, ReconnectMin: time.Second, ReconnectMax: 30 * time.Second}
	if value, ok := os.LookupEnv("ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES"); ok {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES must be a positive integer")
		}
		c.Projector.MaxInFlightSources = n
	}
	if value, ok := os.LookupEnv("ATHENA_TRADER_SYNC_SHUTDOWN_TIMEOUT"); ok {
		c.ShutdownTimeout, err = time.ParseDuration(value)
		if err != nil || c.ShutdownTimeout <= 0 {
			return Config{}, fmt.Errorf("ATHENA_TRADER_SYNC_SHUTDOWN_TIMEOUT requires a positive duration")
		}
	}
	return c, c.Validate()
}
