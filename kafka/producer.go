package kafkaproducer

import (
	"os"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sirupsen/logrus"
)

/*
./kafka-console-consumer --bootstrap-server localhost:9092 --topic test --from-beginning
./kafka-console-producer --broker-list localhost:9092 --topic test
./kafka-topics --list --zookeeper localhost:2181
*/

// Producer stores the main session
type Producer struct {
	Kafkalient *kafka.Producer
}

// Initialize a kafka instance
func (k *Producer) Initialize() {
	brokers := os.Getenv("BROKERS")
	client, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":            brokers,
		"message.max.bytes":            100857600,
		"queue.buffering.max.messages": 1000000,
		//"queue.buffering.max.ms":       5000,
		//"batch.num.messages":           0,
		"log.connection.close": false})
	//	"client.id":                    socket.gethostname(),
	//"default.topic.config": {"acks": "all"}})

	if err != nil {
		logrus.Info("No connection with kafka, ", err)
		os.Exit(1)
	}

	k.Kafkalient = client
}

// PublishMessage send a message to kafka server
func (k *Producer) PublishMessage(topic string, message []byte, partition int32) {
	// Produce messages to topic (asynchronously)
	err := k.Kafkalient.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}, nil)
	if err != nil {
		logrus.Warn("Problem to publish message, ", err)
	}
}
