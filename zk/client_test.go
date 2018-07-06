package zk

import (
	"testing"
)

func TestGozkClient_GetData(t *testing.T) {
	client, err := NewGozkClient([]string{"192.168.1.16:2181"}, "/vpnode", []byte(""))
	if err != nil {
		t.Fatal(err)
	}

	data := <-client.GetData()
	if err != nil {
		t.Logf(string(data))
	}

	t.Logf("1111 %d", len(data))
	select {}
}
