package natsproducer

import (
	"os"

	"github.com/nats-io/go-nats"
	"github.com/sirupsen/logrus"
)

// Producer stores the main session
type Producer struct {
	NatsClient *nats.Conn
}

// Initialize a kafka instance
func (n *Producer) Initialize() {
	nc, err := nats.Connect(os.Getenv("NATS_SERVER"))
	if err != nil {
		logrus.Info("No connection with nats server, ", err)
		os.Exit(1)
	}
	n.NatsClient = nc
}

// PublishMessage send a message to nats server
func (n *Producer) PublishMessage(topic string, message []byte) {
	// Produce messages to topic (asynchronously)
	err := n.NatsClient.Publish(topic, message)
	if err != nil {
		logrus.Warn("Problem to publish message, ", err)
	}
}
