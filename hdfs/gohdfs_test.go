package hdfs

import (
	"testing"
)

func TestHdfs(t *testing.T) {
	client, err := NewHdfsClient("localhost:9000")
	if err != nil {
		t.Fatal(err)
	}

	err = client.WriteFile("/mytest.txt", []byte("test test test"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := client.ReadFile("/mytest.txt")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(data))
}
