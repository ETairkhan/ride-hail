package driven

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ConsumeOptions struct {
	Prefetch     int
	AutoAck      bool
	QueueDurable bool
}

type IDriverBroker interface {
	PublishJSON(ctx context.Context, exchange, routingKey string, msg any) error

	Consume(ctx context.Context, queueName, bindingKey string, opts ConsumeOptions) (<-chan amqp.Delivery, error)

	IsAlive() bool

	Close() error
}
