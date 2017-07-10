package hdfs

import (
	"github.com/colinmarc/hdfs"
	"testing"
)

func TestHdfs(t *testing.T) {
	client, err := hdfs.New("39.108.128.51:9000")
	if err != nil {
		t.Fatal(err)
	}

	fw, err := client.Create(".test.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = fw.Write([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}

	fw.Close()
}
