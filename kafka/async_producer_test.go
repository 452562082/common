package kafka

import (
	"fmt"
	"kuaishangtong/common/utils/log"
	"testing"
	"time"
)

/*
	32945.wav               4621671d71f48520
	32945_raw.wav           76cd44d846444060
	bahxl.wav               1b055aab040fbc90
	dasda.wav               09e7742ece8c6f90
	huang.wav               db316e4357653d40
	tbfaa_A.wav
	tbfaa_B.wav
	test.mp3
	test2.mp3
	test3.mp3
	tlixs_B.wav
	tlixs_C.wav
*/

var data string = `{
"task_id": "%s",
"task_param": {
"task_param_type": "verify",
"task_param_scene": "LLC",
"task_param_object": {
"param_wav_addr": "/home/test/data/32945_raw.wav",
"param_channel": "LEFT",
"param_top_n": 5,
"param_gender": "M",
"param_spk_id": "76cd44d846444060"
}
},
"task_add_time": "%s"
}`

func TestKafkaAsyncProducer(t *testing.T) {
	aproducer, err := NewKafkaAsyncProducer(zkKafkaHosts, "test1")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			select {
			case <-aproducer.Successes():

			case err = <-aproducer.Errors():
				log.Error(err)
				continue
			}
		}
	}()

	for {
		timea := fmt.Sprintf("%d", time.Now().UnixNano())
		log.Debug("send new task", timea)
		//msg := &sarama.ProducerMessage{}
		//msg.Topic = "test1"
		//msg.Key = sarama.ByteEncoder([]byte(timea))
		//msg.Value = sarama.ByteEncoder([]byte(fmt.Sprintf(data, timea, timea)))
		go aproducer.SendByteMessage([]byte(timea), []byte(fmt.Sprintf(data, timea, timea)))
		time.Sleep(2 * time.Second)
	}
}
