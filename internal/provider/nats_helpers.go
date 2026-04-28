package provider

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

func connectNATS(url, creds string) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Timeout(10 * time.Second),
	}
	if creds != "" {
		opts = append(opts, nats.UserCredentialBytes([]byte(creds)))
	}
	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server at %s: %w", url, err)
	}
	return nc, nil
}
