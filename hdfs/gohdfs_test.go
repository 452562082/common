package hdfs

import (
	"testing"
)

func TestHdfsStat(t *testing.T) {
	client, err := NewHdfsClient("192.168.1.185:9000")
	if err != nil {
		t.Fatal(err)
	}

	fi, err := client.client.Stat("/models")
	t.Log(fi.Name())
}

func TestHdfsCopyAllToLocal(t *testing.T) {
	client, err := NewHdfsClient("localhost:9000")
	if err != nil {
		t.Fatal(err)
	}

	err = client.CopyAllFilesToLocal("/models/", "/tmp")
	if err != nil {
		t.Fatal(err)
	}

}

func TestHdfsCopyAllToRemote(t *testing.T) {
	client, err := NewHdfsClient("localhost:9000")
	if err != nil {
		t.Fatal(err)
	}

	err = client.CopyAllFilesToRemote("/usr/local/gowork/src/git.oschina.net/kuaishangtong/common/hdfs/wave", "/wave")
	if err != nil {
		t.Fatal(err)
	}

}
