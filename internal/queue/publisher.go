package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/vitor-labes/pc-scraper/internal/domain"
)

type Publisher struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

func NewPublisher(url, queueName string) (*Publisher, error) {
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

	slog.Info("publisher conectado ao RabbitMQ",
		"queue", queueName,
	)

	return &Publisher{
		conn:      conn,
		channel:   ch,
		queueName: queueName,
	}, nil
}

func (p *Publisher) Publish(ctx context.Context, product domain.Product) error {
	body, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("erro ao serializar produto: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		"",
		p.queueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("erro ao publicar mensagem: %w", err)
	}

	slog.Debug("produto publicado",
		"title", product.Title,
		"price", product.Price,
	)

	return nil
}

func (p *Publisher) Close() error {
	if p.channel != nil {
		if err := p.channel.Close(); err != nil {
			return err
		}
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
