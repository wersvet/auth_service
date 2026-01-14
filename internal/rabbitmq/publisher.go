package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	Publish(ctx context.Context, routingKey string, event any) error
	Close()
}

type AMQPPublisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
}

type NoopPublisher struct {
	reason string
}

func NewPublisher(amqpURL, exchange string) Publisher {
	if strings.TrimSpace(amqpURL) == "" {
		log.Printf("rabbitmq disabled: AMQP_URL is empty")
		return &NoopPublisher{reason: "empty AMQP_URL"}
	}

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Printf("failed to connect to RabbitMQ: %v", err)
		return &NoopPublisher{reason: "connection error"}
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("failed to open RabbitMQ channel: %v", err)
		_ = conn.Close()
		return &NoopPublisher{reason: "channel error"}
	}

	if err := ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Printf("failed to declare exchange %s: %v", exchange, err)
		_ = ch.Close()
		_ = conn.Close()
		return &NoopPublisher{reason: "exchange declare error"}
	}

	return &AMQPPublisher{conn: conn, ch: ch, exchange: exchange}
}

func (p *AMQPPublisher) Publish(ctx context.Context, routingKey string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         payload,
	})
}

func (p *AMQPPublisher) Close() {
	if p == nil {
		return
	}
	if p.ch != nil {
		if err := p.ch.Close(); err != nil {
			log.Printf("failed to close RabbitMQ channel: %v", err)
		}
	}
	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			log.Printf("failed to close RabbitMQ connection: %v", err)
		}
	}
}

func (p *NoopPublisher) Publish(ctx context.Context, routingKey string, event any) error {
	log.Printf("rabbitmq noop publish skipped: %s", p.reason)
	return nil
}

func (p *NoopPublisher) Close() {
	log.Printf("rabbitmq noop publisher closed: %s", p.reason)
}
