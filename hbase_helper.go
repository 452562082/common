package common

import (
	"fmt"

	"git.apache.org/thrift.git/lib/go/thrift"
	"git.oschina.net/kuaishangtong/common/hbase"
)

type HBaseHelper struct {
	*hbase.THBaseServiceClient
}

func NewHBaseHelper(host string, port int) (*HBaseHelper, error) {
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transport, err := thrift.NewTSocket(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	client := hbase.NewTHBaseServiceClientFactory(transport, protocolFactory)
	if err := transport.Open(); err != nil {
		return nil, err
	}

	return &HBaseHelper{
		THBaseServiceClient: client,
	}, nil
}

func (this *HBaseHelper) HBExists(table, rowkey string) (bool, error) {
	return this.Exists([]byte(table), &hbase.TGet{Row: []byte(rowkey)})
}

func (this *HBaseHelper) HBExistsAll(table string, rowkeys []string) ([]bool, error) {
	length := len(rowkeys)
	var tgets []*hbase.TGet
	for i := 0; i < length; i++ {
		tgets = append(tgets, &hbase.TGet{
			Row: []byte(rowkeys[i]),
		})
	}

	return this.ExistsAll([]byte(table), tgets)
}

func (this *HBaseHelper) HBGet(table, rowkey string) (*hbase.TResult_, error) {
	return this.Get([]byte(table), &hbase.TGet{Row: []byte(rowkey)})
}

func (this *HBaseHelper) HBGetMultiple(table string, rowkeys []string) ([]*hbase.TResult_, error) {
	length := len(rowkeys)
	var tgets []*hbase.TGet
	for i := 0; i < length; i++ {
		tgets = append(tgets, &hbase.TGet{
			Row: []byte(rowkeys[i]),
		})
	}

	return this.GetMultiple([]byte(table), tgets)
}

/*
 cvarr := []*hbase.TColumnValue{
        &hbase.TColumnValue{
            Family:    []byte("cf"),
            Qualifier: []byte("title"),
            Value:     []byte("welcome to lesorb.cn")},
        &hbase.TColumnValue{
            Family:    []byte("cf"),
            Qualifier: []byte("content"),
            Value:     []byte("welcome, why are u here!")},
        &hbase.TColumnValue{
            Family:    []byte("cf"),
            Qualifier: []byte("create"),
            Value:     []byte("user5")},
        &hbase.TColumnValue{
            Family:    []byte("cf"),
            Qualifier: []byte("create_time"),
            Value:     []byte("2017-03-21 16:17:26")},
        &hbase.TColumnValue{
            Family:    []byte("cf"),
            Qualifier: []byte("tags"),
            Value:     []byte("welcome,lesorb")}}
*/
func (this *HBaseHelper) HBPut(table, rowkey string, tcValues []*hbase.TColumnValue) error {
	return this.Put([]byte(table), &hbase.TPut{Row: []byte(rowkey), ColumnValues: tcValues})
}

func (this *HBaseHelper) HBPutMultiple(table string, rowkeys []string, tcValues [][]*hbase.TColumnValue) error {
	if len(rowkeys) != len(tcValues) {
		return fmt.Errorf("bad params in rowkeys and TColumnValue")
	}
	length := len(rowkeys)
	var tputArr []*hbase.TPut

	for i := 0; i < length; i++ {
		tputArr = append(tputArr, &hbase.TPut{
			Row:          []byte(rowkeys[i]),
			ColumnValues: tcValues[i],
		})
	}

	return this.PutMultiple([]byte(table), tputArr)
}

func (this *HBaseHelper) HBDeleteSingle(table, rowkey string) error {
	return this.DeleteSingle([]byte(table), &hbase.TDelete{Row: []byte(rowkey)})
}

// 	tColumns := []*hbase.TColumn{
//		&hbase.TColumn{
//			Family:    []byte("cf"),
//			Qualifier: []byte("abc"),
//		},
//	}
func (this *HBaseHelper) HBDeleteWithTColumn(table string,tColumns []*hbase.TColumn) error {
	return this.DeleteSingle([]byte(table), &hbase.TDelete{Columns: tColumns})
}

func (this *HBaseHelper) HBDeleteMultiple(table string, rowkeys []string) ([]*hbase.TDelete, error) {
	length := len(rowkeys)
	var tdels []*hbase.TDelete
	for i := 0; i < length; i++ {
		tdels = append(tdels, &hbase.TDelete{
			Row: []byte(rowkeys[i]),
		})
	}

	return this.DeleteMultiple([]byte(table), tdels)
}

func (this *HBaseHelper) Close() error {
	return this.THBaseServiceClient.Transport.Close()
}
