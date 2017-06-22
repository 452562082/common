package zookeeper

import (
	"testing"
	"time"
)

func TestGozkServer_ServiceRegistry(t *testing.T) {
	server, err := NewGozkServer([]string{"test.shengwenyun.cn:2181"})
	if err != nil {
		t.Fatal(err)
	}
	
	err = server.ServiceRegistry("/asv_servers", "test.shengwenyun.cn:8082", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	
	time.Sleep(5 * time.Hour)
}
