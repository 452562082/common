package hbase

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"testing"
)

var (
	host   string = "39.108.128.51"
	port   string = "9090"
	table  string = "asv_log_info"
	rowkey string = "row_214718asjdk812"
)

func TestNewTHBaseServiceClientFactory(t *testing.T) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocket(host + ":" + port)
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	client := NewTHBaseServiceClientFactory(transport, protocolFactory)
	if err := transport.Open(); err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	isexists, err := client.Exists([]byte(table), &TGet{Row: []byte(rowkey)})
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	log.Infof("rowkey{%s} in table{%s} Exists:%t\n", rowkey, table, isexists)

}
