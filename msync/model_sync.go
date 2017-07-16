package msync

import (
	"fmt"
	"git.oschina.net/kuaishangtong/common/hdfs"
	"git.oschina.net/kuaishangtong/common/kafka"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"os"
	"strings"
)

type ModelSyncer struct {
	hdfsClient *hdfs.HdfsClient

	kafka_sync_topic string
	kafka_sync_group string

	kafka_sync_producer *kafka.KafkaSyncProducer
	kafka_sync_consumer *kafka.KafkaClusterConsumer

	addModelToMemoryfunc func(path string) error
}

func NewModelSyncer( /*hdfs_addrs, hdfs_http_addrs []string,*/ kahosts []string, sync_topic, sync_group string) (*ModelSyncer, error) {
	//client, err := hdfs.DefaultHdfsClient(hdfs_addrs, hdfs_http_addrs, 5)
	//if err != nil {
	//	return nil, err
	//}

	producer, err := kafka.NewKafkaSyncProducer(kahosts, sync_topic)
	if err != nil {
		return nil, err
	}

	consumer, err := kafka.NewKafkaClusterConsumer(kahosts, []string{sync_topic}, sync_group)
	if err != nil {
		return nil, err
	}

	msyncer := &ModelSyncer{
		hdfsClient:          hdfs.DefaultHdfsClient,
		kafka_sync_topic:    sync_topic,
		kafka_sync_group:    sync_group,
		kafka_sync_producer: producer,
		kafka_sync_consumer: consumer,
	}

	go msyncer.loop()

	return msyncer, nil
}

func (ms *ModelSyncer) SetUpdateKey(msid, models []byte) error {
	return ms.kafka_sync_producer.SendByteMessage(msid, models)
}

func (ms *ModelSyncer) loop() {
	for {
		select {
		case msg, ok := <-ms.kafka_sync_consumer.Messages():
			if ok && msg != nil && msg.Value != nil {
				key := string(msg.Key)
				if key != "" {
					fields := strings.Split(key, "#")
					if len(fields) != 2 {
						log.Errorf("ModelSyncer %s get bad key %s", ms.kafka_sync_group, key)
						continue
					}

					if fields[1] != ms.kafka_sync_group {
						switch fields[0] {
						case "add":
							err := ms.DownloadAndSaveNewModel(string(msg.Value))
							if err != nil {
								log.Errorf("ModelSyncer %s download new model failed: %v", ms.kafka_sync_group, err)
							}
						case "del":
							err := ms.DeteleLocalModel(string(msg.Value))
							if err != nil {
								log.Errorf("ModelSyncer %s delete model failed: %v", ms.kafka_sync_group, err)
							} else {
								log.Debugf("modelsyncer %s delete local model %s success", ms.kafka_sync_group, string(msg.Value))
							}
						}
					}
				} else {
					log.Error("bad modelsyncer key, key can not be \"\"")
				}
			}
		case err := <-ms.kafka_sync_consumer.Errors():
			log.Error(err)
		case notice, ok := <-ms.kafka_sync_consumer.Notifications():
			if ok {
				if patitions := notice.Current[ms.kafka_sync_topic]; len(patitions) > 0 {
					log.Debug("ModelSyncer consumer topic:", ms.kafka_sync_topic,
						"patitions:", notice.Current[ms.kafka_sync_topic])
				}
			}
		}
	}
}

// 从HDFS下载增量模型并同步到内存
func (ms *ModelSyncer) DownloadAndSaveNewModel(path string) error {
	log.Debugf("modelsyncer %s download and save model %s", ms.kafka_sync_group, path)
	return ms.hdfsClient.CopyFileToLocal(path, path)
}

// 上传语音文件到HDFS
func (ms *ModelSyncer) UploadNewModel(path string) error {
	err := ms.hdfsClient.CopyFileToRemote(path, path)
	if err != nil {
		return err
	}
	return ms.SetUpdateKey([]byte(fmt.Sprintf("add#%s", ms.kafka_sync_group)), []byte(path))
}

func (ms *ModelSyncer) DeteleRemoteModel(path string) error {
	err := ms.hdfsClient.Remove(path)
	if err != nil {
		return err
	}
	return ms.SetUpdateKey([]byte(fmt.Sprintf("del#%s", ms.kafka_sync_group)), []byte(path))
}

func (ms *ModelSyncer) DeteleLocalModel(path string) error {
	return os.Remove(path)
}
