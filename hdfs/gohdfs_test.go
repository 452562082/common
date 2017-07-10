package hdfs

import (
	"github.com/colinmarc/hdfs"
	"testing"
)

func TestHdfs(t *testing.T) {
	client, err := hdfs.New("izwz9jay6aqdkr9udoehckz:9000")
	if err != nil {
		t.Fatal(err)
	}

	usrname, err := hdfs.Username()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(usrname)

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
