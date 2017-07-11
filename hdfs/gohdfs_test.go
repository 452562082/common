package hdfs

import (
	"fmt"
	"testing"
)

func TestHdfs(t *testing.T) {
	client, err := NewHdfsClient("localhost:9000")
	if err != nil {
		t.Fatal(err)
	}

	err = client.client.MkdirAll("/models/", 0777)

	for i := 0; i < 100; i++ {
		err = client.WriteFile(fmt.Sprintf("/models/m_%05d.ark", i), []byte("test test test"))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestHdfsCopyAllToLocal(t *testing.T) {
	client, err := NewHdfsClient("localhost:9000")
	if err != nil {
		t.Fatal(err)
	}

	err = client.CopyAllFilesToLocal("/models/", "./models/")
	if err != nil {
		t.Fatal(err)
	}

}
