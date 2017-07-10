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

	client, err := hdfs.NewForUser("hdfs://localhost:9000", usrname)
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
