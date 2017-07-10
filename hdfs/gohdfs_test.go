package hdfs

import (
	"github.com/colinmarc/hdfs"
	"testing"
)

func TestHdfs(t *testing.T) {
	client, err := hdfs.New("localhost:9000")
	if err != nil {
		t.Fatal(err)
	}

	fw, err := client.Stat("12.txt")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(fw.Name())
	//
	//_, err = fw.Write([]byte("hello world"))
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//fw.Close()
}
