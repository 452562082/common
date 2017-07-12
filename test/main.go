package main

import (
	"git.oschina.net/kuaishangtong/common/hdfs"
	"git.oschina.net/kuaishangtong/common/utils/log"
)

func main() {
	client, err := hdfs.NewHdfsClient("192.168.1.185:9000")
	if err != nil {
		log.Fatal(err)
	}

	err = client.CopyAllFilesToRemote("/root/asvserver/ivfiles", "/ivfiles")
	if err != nil {
		log.Fatal(err)
	}

	err = client.CopyAllFilesToLocal("/ivfiles", "/root/asvserver/ivfiles_tmp")
	if err != nil {
		log.Fatal(err)
	}
}
