package natsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client implements ports.EventBus using NATS JetStream.
type Client struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
	subs   []jetstream.ConsumeContext
}

// New connects to NATS and ensures the JetStream stream exists.
func New(servers, streamName string) (*Client, error) {
	nc, err := nats.Connect(servers,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				slog.Warn("nats disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			slog.Info("nats reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream init: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{"zenith.>"},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    7 * 24 * time.Hour, // keep events for 7 days
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create stream %s: %w", streamName, err)
	}

	slog.Info("nats connected", "servers", servers, "stream", streamName)
	return &Client{nc: nc, js: js, stream: stream}, nil
}

// Publish sends a platform event to the appropriate NATS subject.
func (c *Client) Publish(ctx context.Context, event *entities.PlatformEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	_, err = c.js.Publish(ctx, string(event.Subject), data)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", event.Subject, err)
	}
	return nil
}

// Subscribe creates a durable consumer for the given subject pattern.
// The consumerName is derived from the subject for uniqueness.
func (c *Client) Subscribe(subject string, handler func(event *entities.PlatformEvent)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	consumerName := "zenith-api-" + sanitize(subject)

	consumer, err := c.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer for %s: %w", subject, err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		var event entities.PlatformEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			slog.Error("failed to unmarshal nats event", "subject", subject, "error", err)
			msg.Nak()
			return
		}
		handler(&event)
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("consume %s: %w", subject, err)
	}

	c.subs = append(c.subs, cc)
	slog.Info("nats subscribed", "subject", subject, "consumer", consumerName)
	return nil
}

// Close drains all subscriptions and closes the NATS connection.
func (c *Client) Close() error {
	for _, sub := range c.subs {
		sub.Stop()
	}
	c.nc.Close()
	return nil
}

// sanitize replaces dots and wildcards with dashes for use in consumer names.
func sanitize(s string) string {
	out := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b == '.' || b == '>' || b == '*' {
			out[i] = '-'
		} else {
			out[i] = b
		}
	}
	return string(out)
}
