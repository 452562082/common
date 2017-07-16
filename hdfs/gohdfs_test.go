package hdfs

import (
	"testing"
)

	func TestHdfsStat(t *testing.T) {
	client, err := NewHdfsClient([]string{"192.168.1.185:9000"}, []string{"192.168.1.185:50070"})
	if err != nil {
		t.Fatal(err)
	}

	fi, err := client.client.Stat("/models")
	t.Log(fi.Name())
}

//func TestHdfsAlive(t *testing.T) {
//	active, err := CheckHDFSAlive("http://39.108.128.51:50070")
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	t.Log(active)
//
//}
//
//func TestHdfsCopyAllToLocal(t *testing.T) {
//	client, err := NewHdfsClient([]string{"192.168.1.185:9000"}, []string{"192.168.1.185:50070"})
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	err = client.CopyAllFilesToLocal("/models/", "/tmp")
//	if err != nil {
//		t.Fatal(err)
//	}
//
//}
//
//func TestHdfsCopyAllToRemote(t *testing.T) {
//	client, err := NewHdfsClient([]string{"192.168.1.185:9000"}, []string{"192.168.1.185:50070"})
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	err = client.CopyAllFilesToRemote("/usr/local/gowork/src/git.oschina.net/kuaishangtong/common/hdfs/wave", "/wave")
//	if err != nil {
//		t.Fatal(err)
//	}
//}
