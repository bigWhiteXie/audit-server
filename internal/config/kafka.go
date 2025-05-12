package config

import (
	"github.com/IBM/sarama"
)

type KafkaConf struct {
	Brokers   []string `json:"brokers"`
	Partition string   `json:"partition,optional"`
}

func InitKafkaClient(kafkaConf KafkaConf) sarama.Client {
	saramaConf := sarama.NewConfig()
	saramaConf.Version = sarama.V2_1_0_0
	saramaConf.Producer.Return.Successes = true
	if kafkaConf.Partition != "" {
		saramaConf.Producer.Partitioner = getPartitioner(kafkaConf.Partition)
	}
	client, err := sarama.NewClient(kafkaConf.Brokers, saramaConf)
	if err != nil {
		panic(err)
	}

	return client
}

func getPartitioner(partitionAlg string) sarama.PartitionerConstructor {
	switch partitionAlg {
	case "hash":
		return sarama.NewHashPartitioner
	case "roundrobin":
		return sarama.NewRoundRobinPartitioner
	case "random":
		return sarama.NewRandomPartitioner
	case "manual":
		return sarama.NewManualPartitioner
	case "consistentHash":
		return sarama.NewConsistentCRCHashPartitioner
	default:
		return sarama.NewRandomPartitioner
	}
}
