package hdfs

import (
	"github.com/colinmarc/hdfs"
	"testing"
)

func TestHdfs(t *testing.T) {
	usrname, err := hdfs.Username()
	if err != nil {
		t.Fatal(err)
	}

	client, err := hdfs.NewForUser("127.0.0.1:9000", usrname)
	if err != nil {
		t.Fatal(err)
	}

	fw, err := client.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = fw.Write([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}

	fw.Close()
}
