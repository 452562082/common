package hbase

import (
	"encoding/binary"
	"git.apache.org/thrift.git/lib/go/thrift"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"testing"
	"time"
)

var (
	host   string = "127.0.0.1"
	port   string = "9090"
	table  string = "asv_log_info"
	rowkey string = "rowkey1111"
)

func TestNewTHBaseServiceExists(t *testing.T) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocketTimeout(host+":"+port, 10*time.Second)
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	log.Debug(transport.Conn())

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

func TestNewTHBaseServiceGet(t *testing.T) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocketTimeout(host+":"+port, 10*time.Second)
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	log.Debug(transport.Conn())

	client := NewTHBaseServiceClientFactory(transport, protocolFactory)
	if err := transport.Open(); err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	value, err := client.Get([]byte(table), &TGet{Row: []byte(rowkey)})
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	log.Infof("rowkey{%s} in table{%s} value :%s", rowkey, table, value.String())
}
func TestNewTHBaseServiceScan(t *testing.T) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocketTimeout(host+":"+port, 10*time.Second)
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	log.Debug(transport.Conn())

	client := NewTHBaseServiceClientFactory(transport, protocolFactory)
	if err := transport.Open(); err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	startrow := make([]byte, 4)
	binary.LittleEndian.PutUint32(startrow, 1)
	stoprow := make([]byte, 4)
	binary.LittleEndian.PutUint32(stoprow, 10)

	r, err := client.GetScannerResults([]byte(table), &TScan{
		StartRow: startrow,
		StopRow:  stoprow,
	}, 100)

	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	for _, v := range r {
		log.Infof("scan in table{%s} %s", table, v.String())
	}

}
