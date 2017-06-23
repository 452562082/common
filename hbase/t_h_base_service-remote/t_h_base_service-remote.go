// Autogenerated by Thrift Compiler (1.0.0-dev)
// DO NOT EDIT UNLESS YOU ARE SURE THAT YOU KNOW WHAT YOU ARE DOING

package main

import (
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"git.oschina.net/kuaishangtong/common/hbase"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func Usage() {
	fmt.Fprintln(os.Stderr, "Usage of ", os.Args[0], " [-h host:port] [-u url] [-f[ramed]] function [arg1 [arg2...]]:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nFunctions:")
	fmt.Fprintln(os.Stderr, "  bool exists(string table, TGet tget)")
	fmt.Fprintln(os.Stderr, "   existsAll(string table,  tgets)")
	fmt.Fprintln(os.Stderr, "  TResult get(string table, TGet tget)")
	fmt.Fprintln(os.Stderr, "   getMultiple(string table,  tgets)")
	fmt.Fprintln(os.Stderr, "  void put(string table, TPut tput)")
	fmt.Fprintln(os.Stderr, "  bool checkAndPut(string table, string row, string family, string qualifier, string value, TPut tput)")
	fmt.Fprintln(os.Stderr, "  void putMultiple(string table,  tputs)")
	fmt.Fprintln(os.Stderr, "  void deleteSingle(string table, TDelete tdelete)")
	fmt.Fprintln(os.Stderr, "   deleteMultiple(string table,  tdeletes)")
	fmt.Fprintln(os.Stderr, "  bool checkAndDelete(string table, string row, string family, string qualifier, string value, TDelete tdelete)")
	fmt.Fprintln(os.Stderr, "  TResult increment(string table, TIncrement tincrement)")
	fmt.Fprintln(os.Stderr, "  TResult append(string table, TAppend tappend)")
	fmt.Fprintln(os.Stderr, "  i32 openScanner(string table, TScan tscan)")
	fmt.Fprintln(os.Stderr, "   getScannerRows(i32 scannerId, i32 numRows)")
	fmt.Fprintln(os.Stderr, "  void closeScanner(i32 scannerId)")
	fmt.Fprintln(os.Stderr, "  void mutateRow(string table, TRowMutations trowMutations)")
	fmt.Fprintln(os.Stderr, "   getScannerResults(string table, TScan tscan, i32 numRows)")
	fmt.Fprintln(os.Stderr, "  THRegionLocation getRegionLocation(string table, string row, bool reload)")
	fmt.Fprintln(os.Stderr, "   getAllRegionLocations(string table)")
	fmt.Fprintln(os.Stderr, "  bool checkAndMutate(string table, string row, string family, string qualifier, TCompareOp compareOp, string value, TRowMutations rowMutations)")
	fmt.Fprintln(os.Stderr)
	os.Exit(0)
}

func main() {
	flag.Usage = Usage
	var host string
	var port int
	var protocol string
	var urlString string
	var framed bool
	var useHttp bool
	var parsedUrl url.URL
	var trans thrift.TTransport
	_ = strconv.Atoi
	_ = math.Abs
	flag.Usage = Usage
	flag.StringVar(&host, "h", "localhost", "Specify host and port")
	flag.IntVar(&port, "p", 9090, "Specify port")
	flag.StringVar(&protocol, "P", "binary", "Specify the protocol (binary, compact, simplejson, json)")
	flag.StringVar(&urlString, "u", "", "Specify the url")
	flag.BoolVar(&framed, "framed", false, "Use framed transport")
	flag.BoolVar(&useHttp, "http", false, "Use http")
	flag.Parse()

	if len(urlString) > 0 {
		parsedUrl, err := url.Parse(urlString)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing URL: ", err)
			flag.Usage()
		}
		host = parsedUrl.Host
		useHttp = len(parsedUrl.Scheme) <= 0 || parsedUrl.Scheme == "http"
	} else if useHttp {
		_, err := url.Parse(fmt.Sprint("http://", host, ":", port))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing URL: ", err)
			flag.Usage()
		}
	}

	cmd := flag.Arg(0)
	var err error
	if useHttp {
		trans, err = thrift.NewTHttpClient(parsedUrl.String())
	} else {
		portStr := fmt.Sprint(port)
		if strings.Contains(host, ":") {
			host, portStr, err = net.SplitHostPort(host)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error with host:", err)
				os.Exit(1)
			}
		}
		trans, err = thrift.NewTSocket(net.JoinHostPort(host, portStr))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error resolving address:", err)
			os.Exit(1)
		}
		if framed {
			trans = thrift.NewTFramedTransport(trans)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating transport", err)
		os.Exit(1)
	}
	defer trans.Close()
	var protocolFactory thrift.TProtocolFactory
	switch protocol {
	case "compact":
		protocolFactory = thrift.NewTCompactProtocolFactory()
		break
	case "simplejson":
		protocolFactory = thrift.NewTSimpleJSONProtocolFactory()
		break
	case "json":
		protocolFactory = thrift.NewTJSONProtocolFactory()
		break
	case "binary", "":
		protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
		break
	default:
		fmt.Fprintln(os.Stderr, "Invalid protocol specified: ", protocol)
		Usage()
		os.Exit(1)
	}
	client := hbase.NewTHBaseServiceClientFactory(trans, protocolFactory)
	if err := trans.Open(); err != nil {
		fmt.Fprintln(os.Stderr, "Error opening socket to ", host, ":", port, " ", err)
		os.Exit(1)
	}

	switch cmd {
	case "exists":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "Exists requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg76 := flag.Arg(2)
		mbTrans77 := thrift.NewTMemoryBufferLen(len(arg76))
		defer mbTrans77.Close()
		_, err78 := mbTrans77.WriteString(arg76)
		if err78 != nil {
			Usage()
			return
		}
		factory79 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt80 := factory79.GetProtocol(mbTrans77)
		argvalue1 := hbase.NewTGet()
		err81 := argvalue1.Read(jsProt80)
		if err81 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		fmt.Print(client.Exists(value0, value1))
		fmt.Print("\n")
		break
	case "existsAll":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "ExistsAll requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg83 := flag.Arg(2)
		mbTrans84 := thrift.NewTMemoryBufferLen(len(arg83))
		defer mbTrans84.Close()
		_, err85 := mbTrans84.WriteString(arg83)
		if err85 != nil {
			Usage()
			return
		}
		factory86 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt87 := factory86.GetProtocol(mbTrans84)
		containerStruct1 := hbase.NewTHBaseServiceExistsAllArgs()
		err88 := containerStruct1.ReadField2(jsProt87)
		if err88 != nil {
			Usage()
			return
		}
		argvalue1 := containerStruct1.Tgets
		value1 := argvalue1
		fmt.Print(client.ExistsAll(value0, value1))
		fmt.Print("\n")
		break
	case "get":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "Get requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg90 := flag.Arg(2)
		mbTrans91 := thrift.NewTMemoryBufferLen(len(arg90))
		defer mbTrans91.Close()
		_, err92 := mbTrans91.WriteString(arg90)
		if err92 != nil {
			Usage()
			return
		}
		factory93 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt94 := factory93.GetProtocol(mbTrans91)
		argvalue1 := hbase.NewTGet()
		err95 := argvalue1.Read(jsProt94)
		if err95 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		fmt.Print(client.Get(value0, value1))
		fmt.Print("\n")
		break
	case "getMultiple":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "GetMultiple requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg97 := flag.Arg(2)
		mbTrans98 := thrift.NewTMemoryBufferLen(len(arg97))
		defer mbTrans98.Close()
		_, err99 := mbTrans98.WriteString(arg97)
		if err99 != nil {
			Usage()
			return
		}
		factory100 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt101 := factory100.GetProtocol(mbTrans98)
		containerStruct1 := hbase.NewTHBaseServiceGetMultipleArgs()
		err102 := containerStruct1.ReadField2(jsProt101)
		if err102 != nil {
			Usage()
			return
		}
		argvalue1 := containerStruct1.Tgets
		value1 := argvalue1
		fmt.Print(client.GetMultiple(value0, value1))
		fmt.Print("\n")
		break
	case "put":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "Put requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg104 := flag.Arg(2)
		mbTrans105 := thrift.NewTMemoryBufferLen(len(arg104))
		defer mbTrans105.Close()
		_, err106 := mbTrans105.WriteString(arg104)
		if err106 != nil {
			Usage()
			return
		}
		factory107 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt108 := factory107.GetProtocol(mbTrans105)
		argvalue1 := hbase.NewTPut()
		err109 := argvalue1.Read(jsProt108)
		if err109 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		fmt.Print(client.Put(value0, value1))
		fmt.Print("\n")
		break
	case "checkAndPut":
		if flag.NArg()-1 != 6 {
			fmt.Fprintln(os.Stderr, "CheckAndPut requires 6 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		argvalue1 := []byte(flag.Arg(2))
		value1 := argvalue1
		argvalue2 := []byte(flag.Arg(3))
		value2 := argvalue2
		argvalue3 := []byte(flag.Arg(4))
		value3 := argvalue3
		argvalue4 := []byte(flag.Arg(5))
		value4 := argvalue4
		arg115 := flag.Arg(6)
		mbTrans116 := thrift.NewTMemoryBufferLen(len(arg115))
		defer mbTrans116.Close()
		_, err117 := mbTrans116.WriteString(arg115)
		if err117 != nil {
			Usage()
			return
		}
		factory118 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt119 := factory118.GetProtocol(mbTrans116)
		argvalue5 := hbase.NewTPut()
		err120 := argvalue5.Read(jsProt119)
		if err120 != nil {
			Usage()
			return
		}
		value5 := argvalue5
		fmt.Print(client.CheckAndPut(value0, value1, value2, value3, value4, value5))
		fmt.Print("\n")
		break
	case "putMultiple":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "PutMultiple requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg122 := flag.Arg(2)
		mbTrans123 := thrift.NewTMemoryBufferLen(len(arg122))
		defer mbTrans123.Close()
		_, err124 := mbTrans123.WriteString(arg122)
		if err124 != nil {
			Usage()
			return
		}
		factory125 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt126 := factory125.GetProtocol(mbTrans123)
		containerStruct1 := hbase.NewTHBaseServicePutMultipleArgs()
		err127 := containerStruct1.ReadField2(jsProt126)
		if err127 != nil {
			Usage()
			return
		}
		argvalue1 := containerStruct1.Tputs
		value1 := argvalue1
		fmt.Print(client.PutMultiple(value0, value1))
		fmt.Print("\n")
		break
	case "deleteSingle":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "DeleteSingle requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg129 := flag.Arg(2)
		mbTrans130 := thrift.NewTMemoryBufferLen(len(arg129))
		defer mbTrans130.Close()
		_, err131 := mbTrans130.WriteString(arg129)
		if err131 != nil {
			Usage()
			return
		}
		factory132 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt133 := factory132.GetProtocol(mbTrans130)
		argvalue1 := hbase.NewTDelete()
		err134 := argvalue1.Read(jsProt133)
		if err134 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		fmt.Print(client.DeleteSingle(value0, value1))
		fmt.Print("\n")
		break
	case "deleteMultiple":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "DeleteMultiple requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg136 := flag.Arg(2)
		mbTrans137 := thrift.NewTMemoryBufferLen(len(arg136))
		defer mbTrans137.Close()
		_, err138 := mbTrans137.WriteString(arg136)
		if err138 != nil {
			Usage()
			return
		}
		factory139 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt140 := factory139.GetProtocol(mbTrans137)
		containerStruct1 := hbase.NewTHBaseServiceDeleteMultipleArgs()
		err141 := containerStruct1.ReadField2(jsProt140)
		if err141 != nil {
			Usage()
			return
		}
		argvalue1 := containerStruct1.Tdeletes
		value1 := argvalue1
		fmt.Print(client.DeleteMultiple(value0, value1))
		fmt.Print("\n")
		break
	case "checkAndDelete":
		if flag.NArg()-1 != 6 {
			fmt.Fprintln(os.Stderr, "CheckAndDelete requires 6 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		argvalue1 := []byte(flag.Arg(2))
		value1 := argvalue1
		argvalue2 := []byte(flag.Arg(3))
		value2 := argvalue2
		argvalue3 := []byte(flag.Arg(4))
		value3 := argvalue3
		argvalue4 := []byte(flag.Arg(5))
		value4 := argvalue4
		arg147 := flag.Arg(6)
		mbTrans148 := thrift.NewTMemoryBufferLen(len(arg147))
		defer mbTrans148.Close()
		_, err149 := mbTrans148.WriteString(arg147)
		if err149 != nil {
			Usage()
			return
		}
		factory150 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt151 := factory150.GetProtocol(mbTrans148)
		argvalue5 := hbase.NewTDelete()
		err152 := argvalue5.Read(jsProt151)
		if err152 != nil {
			Usage()
			return
		}
		value5 := argvalue5
		fmt.Print(client.CheckAndDelete(value0, value1, value2, value3, value4, value5))
		fmt.Print("\n")
		break
	case "increment":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "Increment requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg154 := flag.Arg(2)
		mbTrans155 := thrift.NewTMemoryBufferLen(len(arg154))
		defer mbTrans155.Close()
		_, err156 := mbTrans155.WriteString(arg154)
		if err156 != nil {
			Usage()
			return
		}
		factory157 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt158 := factory157.GetProtocol(mbTrans155)
		argvalue1 := hbase.NewTIncrement()
		err159 := argvalue1.Read(jsProt158)
		if err159 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		fmt.Print(client.Increment(value0, value1))
		fmt.Print("\n")
		break
	case "append":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "Append requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg161 := flag.Arg(2)
		mbTrans162 := thrift.NewTMemoryBufferLen(len(arg161))
		defer mbTrans162.Close()
		_, err163 := mbTrans162.WriteString(arg161)
		if err163 != nil {
			Usage()
			return
		}
		factory164 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt165 := factory164.GetProtocol(mbTrans162)
		argvalue1 := hbase.NewTAppend()
		err166 := argvalue1.Read(jsProt165)
		if err166 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		fmt.Print(client.Append(value0, value1))
		fmt.Print("\n")
		break
	case "openScanner":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "OpenScanner requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg168 := flag.Arg(2)
		mbTrans169 := thrift.NewTMemoryBufferLen(len(arg168))
		defer mbTrans169.Close()
		_, err170 := mbTrans169.WriteString(arg168)
		if err170 != nil {
			Usage()
			return
		}
		factory171 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt172 := factory171.GetProtocol(mbTrans169)
		argvalue1 := hbase.NewTScan()
		err173 := argvalue1.Read(jsProt172)
		if err173 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		fmt.Print(client.OpenScanner(value0, value1))
		fmt.Print("\n")
		break
	case "getScannerRows":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "GetScannerRows requires 2 args")
			flag.Usage()
		}
		tmp0, err174 := (strconv.Atoi(flag.Arg(1)))
		if err174 != nil {
			Usage()
			return
		}
		argvalue0 := int32(tmp0)
		value0 := argvalue0
		tmp1, err175 := (strconv.Atoi(flag.Arg(2)))
		if err175 != nil {
			Usage()
			return
		}
		argvalue1 := int32(tmp1)
		value1 := argvalue1
		fmt.Print(client.GetScannerRows(value0, value1))
		fmt.Print("\n")
		break
	case "closeScanner":
		if flag.NArg()-1 != 1 {
			fmt.Fprintln(os.Stderr, "CloseScanner requires 1 args")
			flag.Usage()
		}
		tmp0, err176 := (strconv.Atoi(flag.Arg(1)))
		if err176 != nil {
			Usage()
			return
		}
		argvalue0 := int32(tmp0)
		value0 := argvalue0
		fmt.Print(client.CloseScanner(value0))
		fmt.Print("\n")
		break
	case "mutateRow":
		if flag.NArg()-1 != 2 {
			fmt.Fprintln(os.Stderr, "MutateRow requires 2 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg178 := flag.Arg(2)
		mbTrans179 := thrift.NewTMemoryBufferLen(len(arg178))
		defer mbTrans179.Close()
		_, err180 := mbTrans179.WriteString(arg178)
		if err180 != nil {
			Usage()
			return
		}
		factory181 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt182 := factory181.GetProtocol(mbTrans179)
		argvalue1 := hbase.NewTRowMutations()
		err183 := argvalue1.Read(jsProt182)
		if err183 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		fmt.Print(client.MutateRow(value0, value1))
		fmt.Print("\n")
		break
	case "getScannerResults":
		if flag.NArg()-1 != 3 {
			fmt.Fprintln(os.Stderr, "GetScannerResults requires 3 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		arg185 := flag.Arg(2)
		mbTrans186 := thrift.NewTMemoryBufferLen(len(arg185))
		defer mbTrans186.Close()
		_, err187 := mbTrans186.WriteString(arg185)
		if err187 != nil {
			Usage()
			return
		}
		factory188 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt189 := factory188.GetProtocol(mbTrans186)
		argvalue1 := hbase.NewTScan()
		err190 := argvalue1.Read(jsProt189)
		if err190 != nil {
			Usage()
			return
		}
		value1 := argvalue1
		tmp2, err191 := (strconv.Atoi(flag.Arg(3)))
		if err191 != nil {
			Usage()
			return
		}
		argvalue2 := int32(tmp2)
		value2 := argvalue2
		fmt.Print(client.GetScannerResults(value0, value1, value2))
		fmt.Print("\n")
		break
	case "getRegionLocation":
		if flag.NArg()-1 != 3 {
			fmt.Fprintln(os.Stderr, "GetRegionLocation requires 3 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		argvalue1 := []byte(flag.Arg(2))
		value1 := argvalue1
		argvalue2 := flag.Arg(3) == "true"
		value2 := argvalue2
		fmt.Print(client.GetRegionLocation(value0, value1, value2))
		fmt.Print("\n")
		break
	case "getAllRegionLocations":
		if flag.NArg()-1 != 1 {
			fmt.Fprintln(os.Stderr, "GetAllRegionLocations requires 1 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		fmt.Print(client.GetAllRegionLocations(value0))
		fmt.Print("\n")
		break
	case "checkAndMutate":
		if flag.NArg()-1 != 7 {
			fmt.Fprintln(os.Stderr, "CheckAndMutate requires 7 args")
			flag.Usage()
		}
		argvalue0 := []byte(flag.Arg(1))
		value0 := argvalue0
		argvalue1 := []byte(flag.Arg(2))
		value1 := argvalue1
		argvalue2 := []byte(flag.Arg(3))
		value2 := argvalue2
		argvalue3 := []byte(flag.Arg(4))
		value3 := argvalue3
		tmp4, err := (strconv.Atoi(flag.Arg(5)))
		if err != nil {
			Usage()
			return
		}
		argvalue4 := hbase.TCompareOp(tmp4)
		value4 := argvalue4
		argvalue5 := []byte(flag.Arg(6))
		value5 := argvalue5
		arg201 := flag.Arg(7)
		mbTrans202 := thrift.NewTMemoryBufferLen(len(arg201))
		defer mbTrans202.Close()
		_, err203 := mbTrans202.WriteString(arg201)
		if err203 != nil {
			Usage()
			return
		}
		factory204 := thrift.NewTSimpleJSONProtocolFactory()
		jsProt205 := factory204.GetProtocol(mbTrans202)
		argvalue6 := hbase.NewTRowMutations()
		err206 := argvalue6.Read(jsProt205)
		if err206 != nil {
			Usage()
			return
		}
		value6 := argvalue6
		fmt.Print(client.CheckAndMutate(value0, value1, value2, value3, value4, value5, value6))
		fmt.Print("\n")
		break
	case "":
		Usage()
		break
	default:
		fmt.Fprintln(os.Stderr, "Invalid function ", cmd)
	}
}