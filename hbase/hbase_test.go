package hbase

import (
	"encoding/binary"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"testing"
	"time"
)

var (
	host   string = "39.108.128.51"
	port   string = "9090"
	table  string = "asv_vpr_info"
	rowkey string = "rowkey1111"
)

func TestNewTHBaseServiceExists(t *testing.T) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocketTimeout(fmt.Sprintf("%s:%s", host, port), 10*time.Second)
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

	log.Infof("rowkey{%s} in table{%s} Exists: %t\n", rowkey, table, isexists)
}

func TestNewTHBaseServiceGet(t *testing.T) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocketTimeout(host+":"+port, 10*time.Second)
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	client := NewTHBaseServiceClientFactory(transport, protocolFactory)
	if err := transport.Open(); err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	 	tColumns := []*TColumn{
			&TColumn{
				Family:    []byte("vpr_spkid"),
				Qualifier: []byte("vpr"),
			},
		}

	value, err := client.Get([]byte(table), &TGet{tColumns})
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	log.Infof("rowkey{%s} in table{%s} Get: %s", rowkey, table, string(value.Row))
}

func TestNewTHBaseServicePut(t *testing.T) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocketTimeout(host+":"+port, 10*time.Second)
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	client := NewTHBaseServiceClientFactory(transport, protocolFactory)
	if err := transport.Open(); err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	cvarr := []*TColumnValue{
		{
			Family:    []byte("log_taskid"),
			Qualifier: []byte("hbase_go_test"),
			Value:     []byte("54321"),
		},
	}

	rowkey2 := "rowkey222"
	temptput := TPut{Row: []byte(rowkey2), ColumnValues: cvarr}

	err = client.Put([]byte(table), &temptput)
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

	log.Infof("rowkey{%s} in table{%s} Put", rowkey2, table)
}

func TestNewTHBaseServiceScan(t *testing.T) {

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocketTimeout(host+":"+port, 10*time.Second)
	if err != nil {
		log.Error(err)
		t.Fatal(err)
	}

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
