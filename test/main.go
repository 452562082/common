package main

import (
	//"git.oschina.net/kuaishangtong/common/hdfs"
	"git.oschina.net/kuaishangtong/common/hdfs"
	"git.oschina.net/kuaishangtong/common/msync"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"io/ioutil"
	"os"
	"time"
)

var kafkaHosts []string = []string{"103.27.5.136:9092"}
var hdfsHosts string = "192.168.1.185:9000"

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

	//TestModelSyncer()
	TestInitSyncModel(hdfsHosts, "/root/asvserver/ivfiles")
}

func TestInitSyncModel(hdfsaddr, modeldir string) {

	if modeldir[len(modeldir)-1] != '/' {
		modeldir += "/"
	}
	hdfsClient, err := hdfs.NewHdfsClient(hdfsaddr)
	if err != nil {
		log.Fatal(err)
	}

	var localmap, hdfsmap map[string]struct{} = make(map[string]struct{}), make(map[string]struct{})

	local_file_infos, err := ioutil.ReadDir(modeldir)
	if err != nil {
		log.Fatal(err)
	}

	for i, v := range local_file_infos {
		if i < 10 {
			log.Debug(modeldir + v.Name())
		}
		localmap[modeldir+v.Name()] = struct{}{}
	}

	log.Infof("catch local ivfiles, count: %d", len(local_file_infos))

	hdfs_file_infos, err := hdfsClient.ReadDir(modeldir)
	if err != nil {
		log.Fatal(err)
	}

	for i, v := range hdfs_file_infos {
		if i < 10 {
			log.Debug(modeldir + v.Name())
		}
		hdfsmap[modeldir+v.Name()] = struct{}{}
	}

	log.Infof("catch hdfs ivfiles, count: %d", len(hdfs_file_infos))

	for k, _ := range hdfsmap {

		if _, ok := localmap[k]; ok {
			delete(localmap, k)
		}

		delete(hdfsmap, k)
	}

	download := len(hdfsmap)

	log.Infof("%d need to download to local", download)

	delete := len(localmap)

	log.Infof("%d need to delete", delete)

	for k, _ := range localmap {
		err := os.Remove(k)
		if err != nil {
			log.Error(err)
		}
		log.Debugf("remove %s", k)
	}

	for k, _ := range hdfsmap {
		err := hdfsClient.CopyFileToLocal(k, k)
		if err != nil {
			log.Error(err)
		}

		log.Debugf("download %s", k)
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
			log.Error(err)
		}
		log.Infof("mSyncer1 SetUpdateKey")

		time.Sleep(3 * time.Second)
		err = mSyncer2.UploadNewModel("/root/asvserver/ivfiles/195ee45733cdca30.ark")
		if err != nil {
			log.Error(err)
		}
		log.Infof("mSyncer2 SetUpdateKey")

		time.Sleep(3 * time.Second)
		err = mSyncer3.UploadNewModel("/root/asvserver/ivfiles/195fb6e963b07420.ark")
		if err != nil {
			log.Error(err)
		}
		log.Infof("mSyncer3 SetUpdateKey")

		time.Sleep(60 * time.Second)
		time.Sleep(3 * time.Second)
		err = mSyncer1.DeteleRemoteModel("/root/asvserver/ivfiles/195e4d3d6a025e60.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("mSyncer1 SetDeleteKey")

		time.Sleep(3 * time.Second)
		err = mSyncer2.DeteleRemoteModel("/root/asvserver/ivfiles/195ee45733cdca30.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("mSyncer2 SetDeleteKey")

		time.Sleep(3 * time.Second)
		err = mSyncer3.DeteleRemoteModel("/root/asvserver/ivfiles/195fb6e963b07420.ark")
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("mSyncer3 SetDeleteKey")

	}
}
