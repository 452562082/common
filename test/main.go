package main

import (
	//"git.oschina.net/kuaishangtong/common/hdfs"
	"git.oschina.net/kuaishangtong/common/hdfs"
	"git.oschina.net/kuaishangtong/common/msync"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"time"
)

var kafkaHosts []string = []string{"103.27.5.136:9092"}
var hdfsHosts []string = []string{"192.168.1.185:9000"}
var hdfsWebHosts []string = []string{"192.168.1.185:50070"}

func main() {
	//client, err := hdfs.NewHdfsClient("192.168.1.185:9000")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//err = client.CopyAllFilesToRemote("/root/asvserver/ivfiles", "/root/asvserver/ivfiles")
	//if err != nil {
	//	log.Fatal(err)
	//}

	//err = client.CopyAllFilesToLocal("/ivfiles", "/root/asvserver/ivfiles_tmp")
	//if err != nil {
	//	log.Fatal(err)
	//}

	log.SetLogFuncCall(true)

	err := TestInitSyncModel(hdfsHosts, hdfsWebHosts, "/root/asvserver/ivfiles")
	if err != nil {
		log.Fatal(err)
	}

	err = hdfs.DefaultHdfsClient.ResetHDFSConnection("izwz9jay6aqdkr9udoehckz:9000")
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("ResetHDFSConnection to %s", "izwz9jay6aqdkr9udoehckz:9000")

	TestModelSyncer()
}

func TestInitSyncModel(hdfs_addrs, hdfs_http_addrs []string, modeldir string) error {
	err := hdfs.InitHDFS(hdfs_addrs, hdfs_http_addrs)
	if err != nil {
		return err
	}
	return hdfs.SyncModel(modeldir)
}

func TestModelSyncer() {
	log.SetLogFuncCall(true)
	log.Debug("new mdoel syncer mSyncer1")

	cb := func(path string) error {
		return nil
	}

	mSyncer1, err := msync.NewModelSyncer( /*hdfsHosts, */ kafkaHosts, "SYNC_MODELS", "Model_Syncer_1", cb)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("new mdoel syncer mSyncer2")
	mSyncer2, err := msync.NewModelSyncer( /*hdfsHosts, */ kafkaHosts, "SYNC_MODELS", "Model_Syncer_2", cb)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("new mdoel syncer mSyncer3")
	mSyncer3, err := msync.NewModelSyncer( /*hdfsHosts, */ kafkaHosts, "SYNC_MODELS", "Model_Syncer_3", cb)
	if err != nil {
		log.Fatal(err)
	}

	//  /root/asvserver/ivfiles/1967e4a6b9fcc570.ark
	//  /root/asvserver/ivfiles/19a299a450379850.ark
	//  /root/asvserver/ivfiles/19de4fb195f59050.ark

	for {
		time.Sleep(3 * time.Second)
		err = mSyncer1.UploadNewModel("/root/asvserver/ivfiles/1967e4a6b9fcc570.ark")
		if err != nil {
			log.Error(err)
		}
		log.Infof("mSyncer1 SetUpdateKey")

		time.Sleep(3 * time.Second)
		err = mSyncer2.UploadNewModel("/root/asvserver/ivfiles/19a299a450379850.ark")
		if err != nil {
			log.Error(err)
		}
		log.Infof("mSyncer2 SetUpdateKey")

		time.Sleep(3 * time.Second)
		err = mSyncer3.UploadNewModel("/root/asvserver/ivfiles/19de4fb195f59050.ark")
		if err != nil {
			log.Error(err)
		}
		log.Infof("mSyncer3 SetUpdateKey")

		time.Sleep(20 * time.Second)
		time.Sleep(3 * time.Second)
		err = mSyncer1.DeteleRemoteModel("/root/asvserver/ivfiles/1967e4a6b9fcc570.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("mSyncer1 SetDeleteKey")

		time.Sleep(3 * time.Second)
		err = mSyncer2.DeteleRemoteModel("/root/asvserver/ivfiles/19a299a450379850.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("mSyncer2 SetDeleteKey")

		time.Sleep(3 * time.Second)
		err = mSyncer3.DeteleRemoteModel("/root/asvserver/ivfiles/19de4fb195f59050.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("mSyncer3 SetDeleteKey")

	}
}
