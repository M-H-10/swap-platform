// Package queue connects the server to RabbitMQ: it sends messages to a
// queue and hands the messages of another queue to a Go function.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
)

// Handler processes the body of one message. Returning an error drops the
// message (it is logged), so one broken message cannot block the queue.
type Handler func(ctx context.Context, body []byte) error

// Client keeps one connection to RabbitMQ and reconnects by itself when
// RabbitMQ restarts or is started after the server.
type Client struct {
	url       string
	queues    []string
	consumers map[string]Handler

	mu sync.Mutex
	ch *amqp.Channel // the channel used for publishing; nil while disconnected
}

// New prepares a client. Nothing connects until Run is called.
// queues are created in RabbitMQ (if missing) on every connection.
func New(url string, queues ...string) *Client {
	return &Client{url: url, queues: queues, consumers: map[string]Handler{}}
}

// Consume registers a handler for a queue. Call it before Run.
func (c *Client) Consume(queue string, h Handler) {
	c.consumers[queue] = h
}

// Run connects and keeps the connection alive until ctx is cancelled.
// It blocks, so start it with: go client.Run(ctx)
func (c *Client) Run(ctx context.Context) {
	for {
		err := c.session(ctx)
		if ctx.Err() != nil {
			return // the server is shutting down
		}
		log.Printf("rabbitmq: %v (retrying in 5s)", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// session is one connection: it declares the queues, starts the consumers
// and waits until the connection closes or the server stops.
func (c *Client) session(ctx context.Context) error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	pub, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	for _, q := range c.queues {
		// durable = true: the queue survives a RabbitMQ restart.
		if _, err := pub.QueueDeclare(q, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare queue %s: %w", q, err)
		}
	}

	for q, h := range c.consumers {
		ch, err := conn.Channel()
		if err != nil {
			return fmt.Errorf("open channel: %w", err)
		}
		if err := ch.Qos(10, 0, false); err != nil { // at most 10 unconfirmed messages at a time
			return fmt.Errorf("set qos: %w", err)
		}
		// autoAck = false: a message is removed only after Ack, so nothing
		// is lost if the server stops in the middle of processing.
		deliveries, err := ch.Consume(q, "", false, false, false, false, nil)
		if err != nil {
			return fmt.Errorf("consume %s: %w", q, err)
		}
		go handle(ctx, q, deliveries, h)
	}

	c.setChannel(pub)
	defer c.setChannel(nil)
	log.Println("connected to RabbitMQ")

	closed := conn.NotifyClose(make(chan *amqp.Error, 1))
	select {
	case <-ctx.Done():
		return nil
	case e := <-closed:
		return fmt.Errorf("connection lost: %v", e)
	}
}

func handle(ctx context.Context, queue string, deliveries <-chan amqp.Delivery, h Handler) {
	for d := range deliveries { // the loop ends when the connection closes
		if err := h(ctx, d.Body); err != nil {
			log.Printf("rabbitmq: message from %s dropped: %v", queue, err)
			_ = d.Nack(false, false) // false = do not put it back in the queue
			continue
		}
		_ = d.Ack(false)
	}
}

func (c *Client) setChannel(ch *amqp.Channel) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ch = ch
}

// Publish sends msg as JSON to a queue. The message is persistent: it
// waits in RabbitMQ until the ML service reads it, even across restarts.
func (c *Client) Publish(ctx context.Context, queue string, msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encode message: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ch == nil {
		return fmt.Errorf("%w: not connected to RabbitMQ", domain.ErrUnavailable)
	}
	err = c.ch.PublishWithContext(ctx, "", queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("%w: publish: %v", domain.ErrUnavailable, err)
	}
	return nil
}

// Connected reports whether the client is connected right now. Short
// programs (like cmd/seed) use it to wait for the connection before they
// publish anything.
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ch != nil
}
