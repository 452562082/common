package hdfs

import (
	"kuaishangtong/common/utils/log"
	"testing"
)

func TestHdfsStat(t *testing.T) {
	log.SetLogFuncCall(true)
	client, err := NewHdfsClient(
		[]string{"192.168.1.16:9000"},
		[]string{"192.168.1.16:50070"}, "hadoop", 3)
	if err != nil {
		t.Fatal(err)
	}

	err = client.client.Rename("/tmp/asv/testnode/0a67793c238f6ba0.ark", "/tmp/asv/testnode1/0a67793c238f6ba0.ark")
	if err != nil {
		t.Fatal(err)
	}

}
