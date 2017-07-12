package main

import (
	"asvserver/msync"
	"git.oschina.net/kuaishangtong/common/hdfs"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"time"
)

var kafkaHosts []string = []string{"103.27.5.136:9092"}
var hdfsHosts string = "192.168.1.185:9000"

func main() {
	client, err := hdfs.NewHdfsClient("192.168.1.185:9000")
	if err != nil {
		log.Fatal(err)
	}

	//err = client.CopyAllFilesToRemote("/root/asvserver/ivfiles", "/ivfiles")
	//if err != nil {
	//	log.Fatal(err)
	//}

	err = client.CopyAllFilesToLocal("/ivfiles", "/root/asvserver/ivfiles_tmp")
	if err != nil {
		log.Fatal(err)
	}
}

func TestModelSyncer() {
	log.SetLogFuncCall(true)
	log.Debug("new mdoel syncer mSyncer1")
	mSyncer1, err := msync.NewModelSyncer(hdfsHosts, kafkaHosts, "SYNC_MODELS", "Model_Syncer_1")
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("new mdoel syncer mSyncer2")
	mSyncer2, err := msync.NewModelSyncer(hdfsHosts, kafkaHosts, "SYNC_MODELS", "Model_Syncer_2")
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("new mdoel syncer mSyncer3")
	mSyncer3, err := msync.NewModelSyncer(hdfsHosts, kafkaHosts, "SYNC_MODELS", "Model_Syncer_3")
	if err != nil {
		log.Fatal(err)
	}

	//  /root/asvserver/ivfiles/195e4d3d6a025e60.ark
	//  /root/asvserver/ivfiles/195ee45733cdca30.ark
	//  /root/asvserver/ivfiles/195fb6e963b07420.ark

	for {
		time.Sleep(3 * time.Second)
		err = mSyncer1.UploadNewModel("/root/asvserver/ivfiles/195e4d3d6a025e60.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("mSyncer1 SetUpdateKey")

		time.Sleep(3 * time.Second)
		err = mSyncer2.UploadNewModel("/root/asvserver/ivfiles/195ee45733cdca30.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("mSyncer2 SetUpdateKey")

		time.Sleep(3 * time.Second)
		err = mSyncer3.UploadNewModel("/root/asvserver/ivfiles/195fb6e963b07420.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("mSyncer3 SetUpdateKey")


		time.Sleep(60 * time.Second)
		time.Sleep(3 * time.Second)
		err = mSyncer1.DeteleRemoteModel("/root/asvserver/ivfiles/195e4d3d6a025e60.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("mSyncer1 SetDeleteKey")

		time.Sleep(3 * time.Second)
		err = mSyncer2.DeteleRemoteModel("/root/asvserver/ivfiles/195ee45733cdca30.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("mSyncer2 SetDeleteKey")

		time.Sleep(3 * time.Second)
		err = mSyncer3.DeteleRemoteModel("/root/asvserver/ivfiles/195fb6e963b07420.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("mSyncer3 SetDeleteKey")

	}
}
