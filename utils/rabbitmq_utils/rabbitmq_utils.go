package rabbitmq_utils

import (
	"github.com/streadway/amqp"
	"log"
)

var rabbitConn *amqp.Connection
var rabbitChannel *amqp.Channel

func InitRabbitMQ(brokerAddr string) {
	var err error
	rabbitConn, err = amqp.Dial(brokerAddr)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}

	rabbitChannel, err = rabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %s", err)
	}

	_, err = rabbitChannel.QueueDeclare(
		"my_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %s", err)
	}
}

var PublishToQueue = func(message string) {
	if rabbitChannel == nil {
		log.Println("RabbitMQ channel not initialized")
		return
	}

	err := rabbitChannel.Publish(
		"", "my_queue", false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)
	if err != nil {
		log.Printf("Failed to publish message: %s", err)
	}
}
