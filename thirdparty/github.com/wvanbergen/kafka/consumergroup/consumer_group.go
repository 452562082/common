package consumergroup

import (
	"errors"
	"sync"
	"time"

	"git.oschina.net/kuaishangtong/common/log"
	"git.oschina.net/kuaishangtong/common/thirdparty/github.com/Shopify/sarama"
	"git.oschina.net/kuaishangtong/common/thirdparty/github.com/wvanbergen/kazoo-go"
)

var (
	AlreadyClosing = errors.New("The consumer group is already shutting down.")
)

type Config struct {
	*sarama.Config

	Zookeeper *kazoo.Config

	Offsets struct {
		Initial           int64         // The initial offset method to use if the consumer has no previously stored offset. Must be either sarama.OffsetOldest (default) or sarama.OffsetNewest.
		ProcessingTimeout time.Duration // Time to wait for all the offsets for a partition to be processed after stopping to consume from it. Defaults to 1 minute.
		CommitInterval    time.Duration // The interval between which the processed offsets are commited.
		ResetOffsets      bool          // Resets the offsets for the consumergroup so that it won't resume from where it left off previously.
	}
}

func NewConfig() *Config {
	config := &Config{}
	config.Config = sarama.NewConfig()
	config.Zookeeper = kazoo.NewConfig()
	config.Offsets.Initial = sarama.OffsetOldest
	config.Offsets.ProcessingTimeout = 60 * time.Second
	config.Offsets.CommitInterval = 10 * time.Second

	return config
}

func (cgc *Config) Validate() error {
	if cgc.Zookeeper.Timeout <= 0 {
		return sarama.ConfigurationError("ZookeeperTimeout should have a duration > 0")
	}

	if cgc.Offsets.CommitInterval <= 0 {
		return sarama.ConfigurationError("CommitInterval should have a duration > 0")
	}

	if cgc.Offsets.Initial != sarama.OffsetOldest && cgc.Offsets.Initial != sarama.OffsetNewest {
		return errors.New("Offsets.Initial should be sarama.OffsetOldest or sarama.OffsetNewest.")
	}

	if cgc.Config != nil {
		if err := cgc.Config.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// The ConsumerGroup type holds all the information for a consumer that is part
// of a consumer group. Call JoinConsumerGroup to start a consumer.
type ConsumerGroup struct {
	config *Config

	consumer sarama.Consumer
	kazoo    *kazoo.Kazoo
	group    *kazoo.Consumergroup
	instance *kazoo.ConsumergroupInstance

	wg             sync.WaitGroup
	singleShutdown sync.Once

	messages chan *sarama.ConsumerMessage
	errors   chan *sarama.ConsumerError
	stopper  chan struct{}

	consumers kazoo.ConsumergroupInstanceList

	offsetManager OffsetManager
}

// Connects to a consumer group, using Zookeeper for auto-discovery
func JoinConsumerGroup(name string, topics []string, zookeeper []string, config *Config) (cg *ConsumerGroup, err error) {

	if name == "" {
		return nil, sarama.ConfigurationError("Empty consumergroup name")
	}

	if len(topics) == 0 {
		return nil, sarama.ConfigurationError("No topics provided")
	}

	if len(zookeeper) == 0 {
		return nil, errors.New("You need to provide at least one zookeeper node address!")
	}

	if config == nil {
		config = NewConfig()
	}
	config.ClientID = name

	// Validate configuration
	if err = config.Validate(); err != nil {
		return
	}

	var kz *kazoo.Kazoo
	if kz, err = kazoo.NewKazoo(zookeeper, config.Zookeeper); err != nil {
		return
	}

	brokers, err := kz.BrokerList()
	if err != nil {
		return nil, err
	}

	group := kz.Consumergroup(name)

	if config.Offsets.ResetOffsets {
		err = group.ResetOffsets()
		if err != nil {
			log.Errorf("%s FAILED to reset offsets of consumergroup: %s", cg.group.Name, err)
			kz.Close()
			return
		}
	}

	instance := group.NewInstance()

	var consumer sarama.Consumer
	if consumer, err = sarama.NewConsumer(brokers, config.Config); err != nil {
		kz.Close()
		return
	}

	cg = &ConsumerGroup{
		config:   config,
		consumer: consumer,

		kazoo:    kz,
		group:    group,
		instance: instance,

		messages: make(chan *sarama.ConsumerMessage, config.ChannelBufferSize),
		errors:   make(chan *sarama.ConsumerError, config.ChannelBufferSize),
		stopper:  make(chan struct{}),
	}

	// Register consumer group
	if exists, err := cg.group.Exists(); err != nil {
		log.Errorf("%s FAILED to check for existence of consumergroup: %s", cg.group.Name, err)
		_ = consumer.Close()
		_ = kz.Close()
		return nil, err
	} else if !exists {
		log.Infof("Consumergroup `%s` does not yet exists, creating...", cg.group.Name)
		if err := cg.group.Create(); err != nil {
			log.Errorf("%s FAILED to create consumergroup in Zookeeper: %s", cg.group.Name, err)
			_ = consumer.Close()
			_ = kz.Close()
			return nil, err
		}
	}

	// Register itself with zookeeper
	if err := cg.instance.Register(topics); err != nil {
		log.Errorf("%s FAILED to register consumer instance: %s!", cg.group.Name, err)
		return nil, err
	} else {
		log.Infof("%s Consumer instance registered (%s).", cg.group.Name, cg.instance.ID)
	}

	offsetConfig := OffsetManagerConfig{CommitInterval: config.Offsets.CommitInterval}
	cg.offsetManager = NewZookeeperOffsetManager(cg, &offsetConfig)

	go cg.topicListConsumer(topics)

	return
}

// Returns a channel that you can read to obtain events from Kafka to process.
func (cg *ConsumerGroup) Messages() <-chan *sarama.ConsumerMessage {
	return cg.messages
}

// Returns a channel that you can read to obtain events from Kafka to process.
func (cg *ConsumerGroup) Errors() <-chan *sarama.ConsumerError {
	return cg.errors
}

func (cg *ConsumerGroup) Closed() bool {
	return cg.instance == nil
}

func (cg *ConsumerGroup) Close() error {
	shutdownError := AlreadyClosing
	cg.singleShutdown.Do(func() {
		defer cg.kazoo.Close()

		shutdownError = nil

		close(cg.stopper)
		cg.wg.Wait()

		if err := cg.offsetManager.Close(); err != nil {
			log.Errorf("%s FAILED closing the offset manager: %s", cg.group.Name, err)
		}

		if shutdownError = cg.instance.Deregister(); shutdownError != nil {
			log.Errorf("%s FAILED deregistering consumer instance: %s", cg.group.Name, shutdownError)
		} else {
			log.Infof("%s Deregistered consumer instance %s", cg.group.Name, cg.instance.ID)
		}

		if shutdownError = cg.consumer.Close(); shutdownError != nil {
			log.Errorf("%s FAILED closing the Sarama client: %s", cg.group.Name, shutdownError)
		}

		close(cg.messages)
		close(cg.errors)
		cg.instance = nil
	})

	return shutdownError
}

//func (cg *ConsumerGroup) Logf(,format string, args ...interface{}) {
//	var identifier string
//	if cg.instance == nil {
//		identifier = "(defunct)"
//	} else {
//		identifier = cg.instance.ID[len(cg.instance.ID)-12:]
//	}
//	log.Infof("[%s/%s] %s", cg.group.Name, identifier, fmt.Sprintf(format, args...))
//}

func (cg *ConsumerGroup) InstanceRegistered() (bool, error) {
	return cg.instance.Registered()
}

func (cg *ConsumerGroup) CommitUpto(message *sarama.ConsumerMessage) error {
	cg.offsetManager.MarkAsProcessed(message.Topic, message.Partition, message.Offset)
	return nil
}

func (cg *ConsumerGroup) topicListConsumer(topics []string) {
	for {
		select {
		case <-cg.stopper:
			return
		default:
		}

		consumers, consumerChanges, err := cg.group.WatchInstances()
		if err != nil {
			log.Errorf("%s FAILED to get list of registered consumer instances: %s",
				cg.group.Name, err)
			return
		}

		cg.consumers = consumers
		log.Infof("%s Currently registered consumers: %d", cg.group.Name, len(cg.consumers))

		stopper := make(chan struct{})

		for _, topic := range topics {
			cg.wg.Add(1)
			go cg.topicConsumer(topic, cg.messages, cg.errors, stopper)
		}

		select {
		case <-cg.stopper:
			close(stopper)
			return

		case <-consumerChanges:
			log.Warnf("%s Triggering rebalance due to consumer list change", cg.group.Name)
			close(stopper)
			cg.wg.Wait()
		}
	}
}

func (cg *ConsumerGroup) topicConsumer(topic string, messages chan<- *sarama.ConsumerMessage, errors chan<- *sarama.ConsumerError, stopper <-chan struct{}) {
	defer cg.wg.Done()

	select {
	case <-stopper:
		return
	default:
	}

	log.Infof("%s Started topic %s consumer", cg.group.Name, topic)

	// Fetch a list of partition IDs
	partitions, err := cg.kazoo.Topic(topic).Partitions()
	if err != nil {
		log.Errorf("%s FAILED to get list of partitions from %s: %s", cg.group.Name, topic, err)
		cg.errors <- &sarama.ConsumerError{
			Topic:     topic,
			Partition: -1,
			Err:       err,
		}
		return
	}

	partitionLeaders, err := retrievePartitionLeaders(partitions)
	if err != nil {
		log.Errorf("%s FAILED to get leaders of partitions from %s: %s", cg.group.Name, topic, err)
		cg.errors <- &sarama.ConsumerError{
			Topic:     topic,
			Partition: -1,
			Err:       err,
		}
		return
	}

	dividedPartitions := dividePartitionsBetweenConsumers(cg.consumers, partitionLeaders)
	myPartitions := dividedPartitions[cg.instance.ID]
	log.Infof("%s %s :: Claiming %d of %d partitions",
		cg.group.Name, topic, len(myPartitions), len(partitionLeaders))

	// Consume all the assigned partitions
	var wg sync.WaitGroup
	for _, pid := range myPartitions {

		wg.Add(1)
		go cg.partitionConsumer(topic, pid.ID, messages, errors, &wg, stopper)
	}

	wg.Wait()
	log.Warnf("%s Stopped topic %s consumer", cg.group.Name, topic)
}

// Consumes a partition
func (cg *ConsumerGroup) partitionConsumer(topic string, partition int32, messages chan<- *sarama.ConsumerMessage, errors chan<- *sarama.ConsumerError, wg *sync.WaitGroup, stopper <-chan struct{}) {
	defer wg.Done()

	select {
	case <-stopper:
		return
	default:
	}

	for maxRetries, tries := 3, 0; tries < maxRetries; tries++ {
		if err := cg.instance.ClaimPartition(topic, partition); err == nil {
			break
		} else if err == kazoo.ErrPartitionClaimedByOther && tries+1 < maxRetries {
			time.Sleep(1 * time.Second)
		} else {
			log.Errorf("%s %s/%d :: FAILED to claim the partition: %s", cg.group.Name, topic, partition, err)
			return
		}
	}
	defer cg.instance.ReleasePartition(topic, partition)

	nextOffset, err := cg.offsetManager.InitializePartition(topic, partition)
	if err != nil {
		log.Errorf("%s %s/%d :: FAILED to determine initial offset: %s",
			cg.group.Name, topic, partition, err)
		return
	}

	if nextOffset >= 0 {
		log.Infof("%s %s/%d :: Partition consumer starting at offset %d.",
			cg.group.Name, topic, partition, nextOffset)
	} else {
		nextOffset = cg.config.Offsets.Initial
		if nextOffset == sarama.OffsetOldest {
			log.Infof("%s %s/%d :: Partition consumer starting at the oldest available offset.",
				cg.group.Name, topic, partition)
		} else if nextOffset == sarama.OffsetNewest {
			log.Infof("%s %s/%d :: Partition consumer listening for new messages only.",
				cg.group.Name, topic, partition)
		}
	}

	consumer, err := cg.consumer.ConsumePartition(topic, partition, nextOffset)
	if err == sarama.ErrOffsetOutOfRange {
		log.Errorf("%s %s/%d :: Partition consumer offset out of Range.", cg.group.Name, topic, partition)
		// if the offset is out of range, simplistically decide whether to use OffsetNewest or OffsetOldest
		// if the configuration specified offsetOldest, then switch to the oldest available offset, else
		// switch to the newest available offset.
		if cg.config.Offsets.Initial == sarama.OffsetOldest {
			nextOffset = sarama.OffsetOldest
			log.Infof("%s %s/%d :: Partition consumer offset reset to oldest available offset.",
				cg.group.Name, topic, partition)
		} else {
			nextOffset = sarama.OffsetNewest
			log.Infof("%s %s/%d :: Partition consumer offset reset to newest available offset.",
				cg.group.Name, topic, partition)
		}
		// retry the consumePartition with the adjusted offset
		consumer, err = cg.consumer.ConsumePartition(topic, partition, nextOffset)
	}
	if err != nil {
		log.Errorf("%s %s/%d :: FAILED to start partition consumer: %s", topic, partition, err)
		return
	}
	defer consumer.Close()

	err = nil
	var lastOffset int64 = -1 // aka unknown
partitionConsumerLoop:
	for {
		select {
		case <-stopper:
			break partitionConsumerLoop

		case err := <-consumer.Errors():
			for {
				select {
				case errors <- err:
					continue partitionConsumerLoop

				case <-stopper:
					break partitionConsumerLoop
				}
			}

		case message := <-consumer.Messages():
			for {
				select {
				case <-stopper:
					break partitionConsumerLoop

				case messages <- message:
					lastOffset = message.Offset
					continue partitionConsumerLoop
				}
			}
		}
	}

	log.Infof("%s %s/%d :: Stopping partition consumer at offset %d", topic, cg.group.Name, partition, lastOffset)
	if err := cg.offsetManager.FinalizePartition(topic, partition, lastOffset, cg.config.Offsets.ProcessingTimeout); err != nil {
		log.Errorf("cg.group.Name, %s/%d err: %s", cg.group.Name, topic, partition, err)
	}
}
