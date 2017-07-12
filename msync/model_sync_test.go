package msync

import (
	"git.oschina.net/kuaishangtong/common/utils/log"
	"testing"
	"time"
)

var kafkaHosts []string = []string{"103.27.5.136:9092"}
var hdfsHosts string = "localhost:9000"

func TestModelSyncer(t *testing.T) {
	log.SetLogFuncCall(true)
	log.Debug("new mdoel syncer mSyncer1")
	mSyncer1, err := NewModelSyncer(hdfsHosts, kafkaHosts, "SYNC_MODELS", "Model_Syncer_1")
	if err != nil {
		t.Fatal(err)
	}

	log.Debug("new mdoel syncer mSyncer2")
	mSyncer2, err := NewModelSyncer(hdfsHosts, kafkaHosts, "SYNC_MODELS", "Model_Syncer_2")
	if err != nil {
		t.Fatal(err)
	}

	log.Debug("new mdoel syncer mSyncer3")
	mSyncer3, err := NewModelSyncer(hdfsHosts, kafkaHosts, "SYNC_MODELS", "Model_Syncer_3")
	if err != nil {
		t.Fatal(err)
	}

	mSyncer1.DeteleModel("/mSyncer1.dat")
	mSyncer2.DeteleModel("/mSyncer2.dat")
	mSyncer3.DeteleModel("/mSyncer3.dat")

	for {
		time.Sleep(3 * time.Second)
		err = mSyncer1.UploadNewModel("/mSyncer1.dat")
		if err != nil {
			t.Fatal(err)
		}
		log.Debug("mSyncer1 SetUpdateKey")

		time.Sleep(3 * time.Second)
		err = mSyncer2.UploadNewModel("/mSyncer2.dat")
		if err != nil {
			t.Fatal(err)
		}
		log.Debug("mSyncer2 SetUpdateKey")

		time.Sleep(3 * time.Second)
		err = mSyncer3.UploadNewModel("/mSyncer3.dat")
		if err != nil {
			t.Fatal(err)
		}
		log.Debug("mSyncer3 SetUpdateKey")
	}
}
