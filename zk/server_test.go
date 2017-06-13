package zk

import (
	"testing"
	"time"
)

func TestGozkServer_ServiceRegistry(t *testing.T) {
	server, err := NewGozkServer([]string{"127.0.0.1:2181"})
	if err != nil {
		t.Fatal(err)
	}
	
	err = server.ServiceRegistry("/good/bad/v1/v2", "/127.0.0.1:8111/", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	
	time.Sleep(5 * time.Second)
}
