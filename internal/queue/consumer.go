package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/vitor-labes/pc-scraper/internal/domain"
)

type MessageHandler func(context.Context, domain.Product) error

type Consumer struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
	handler   MessageHandler
}

func NewConsumer(url, queueName string, handler MessageHandler) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar no RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("falha ao abrir canal: %w", err)
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("falha ao declarar fila: %w", err)
	}

	// Process once
	err = ch.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("falha ao configurar QoS: %w", err)
	}

	slog.Info("consumer conectado ao RabbitMQ",
		"queue", queueName,
	)

	return &Consumer{
		conn:      conn,
		channel:   ch,
		queueName: queueName,
		handler:   handler,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("falha ao registrar consumer: %w", err)
	}

	slog.Info("aguardando mensagens...", "queue", c.queueName)

	for {
		select {
		case <-ctx.Done():
			slog.Info("consumer encerrado pelo contexto")
			return ctx.Err()

		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("canal de mensagens fechado")
			}

			if err := c.processMessage(ctx, msg); err != nil {
				slog.Error("erro ao processar mensagem",
					"error", err,
					"body", string(msg.Body),
				)
				// Requeue
				msg.Nack(false, true)
			} else {
				msg.Ack(false)
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg amqp.Delivery) error {
	var product domain.Product
	if err := json.Unmarshal(msg.Body, &product); err != nil {
		return fmt.Errorf("erro ao deserializar mensagem: %w", err)
	}

	slog.Info("processando produto",
		"title", product.Title,
		"price", product.Price,
		"category", product.Category,
	)

	if err := c.handler(ctx, product); err != nil {
		return fmt.Errorf("erro no handler: %w", err)
	}

	return nil
}

func (c *Consumer) Close() error {
	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			return err
		}
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
