package common

import (
	"fmt"

	"git.apache.org/thrift.git/lib/go/thrift"
	"git.oschina.net/kuaishangtong/common/hbase"
	"git.oschina.net/kuaishangtong/common/utils"
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
	return this.Exists(utils.S2B(table), &hbase.TGet{Row: utils.S2B(rowkey)})
}

func (this *HBaseHelper) HBExistsAll(table string, rowkeys []string) ([]bool, error) {
	length := len(rowkeys)
	var tgets []*hbase.TGet
	for i := 0; i < length; i++ {
		tgets = append(tgets, &hbase.TGet{
			Row: utils.S2B(rowkeys[i]),
		})
	}

	return this.ExistsAll(utils.S2B(table), tgets)
}

func (this *HBaseHelper) HBGet(table, rowkey string) (*hbase.TResult_, error) {
	return this.Get(utils.S2B(table), &hbase.TGet{Row: utils.S2B(rowkey)})
}

func (this *HBaseHelper) HBGetMultiple(table string, rowkeys []string) ([]*hbase.TResult_, error) {
	length := len(rowkeys)
	var tgets []*hbase.TGet
	for i := 0; i < length; i++ {
		tgets = append(tgets, &hbase.TGet{
			Row: utils.S2B(rowkeys[i]),
		})
	}

	return this.GetMultiple(utils.S2B(table), tgets)
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
	return this.Put(utils.S2B(table), &hbase.TPut{Row: utils.S2B(rowkey), ColumnValues: tcValues})
}

func (this *HBaseHelper) HBPutMultiple(table string, rowkeys []string, tcValues [][]*hbase.TColumnValue) error {
	if len(rowkeys) != len(tcValues) {
		return fmt.Errorf("bad params in rowkeys and TColumnValue")
	}
	length := len(rowkeys)
	var tputArr []*hbase.TPut

	for i := 0; i < length; i++ {
		tputArr = append(tputArr, &hbase.TPut{
			Row:          utils.S2B(rowkeys[i]),
			ColumnValues: tcValues[i],
		})
	}

	return this.PutMultiple(utils.S2B(table), tputArr)
}

func (this *HBaseHelper) HBDeleteSingle(table, rowkey string) error {
	return this.DeleteSingle(utils.S2B(table), &hbase.TDelete{Row: utils.S2B(rowkey)})
}

func (this *HBaseHelper) HBDeleteMultiple(table string, rowkeys []string) ([]*hbase.TDelete, error) {
	length := len(rowkeys)
	var tdels []*hbase.TDelete
	for i := 0; i < length; i++ {
		tdels = append(tdels, &hbase.TDelete{
			Row: utils.S2B(rowkeys[i]),
		})
	}

	return this.DeleteMultiple(utils.S2B(table), tdels)
}
