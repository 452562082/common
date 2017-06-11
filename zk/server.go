package zk

import (
	"fmt"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"
	"unsafe"

	"tproxy/utils/log"
	"tproxy/zk/gozk"
)

var SessionTimeout int = 100

type GozkServer struct {
	conn *gozk.Conn

	lock *sync.RWMutex
}

func NewGozkServer(zkhosts []string) (*GozkServer, error) {
	conn, _, err := gozk.Connect(zkhosts, time.Duration(SessionTimeout)*time.Millisecond)
	if err != nil {
		return nil, err
	}

	return &GozkServer{
		conn: conn,
		lock: &sync.RWMutex{},
	}, nil
}

func (gzc *GozkServer) String() string {
	return fmt.Sprintf("go-zookeeper Server sid[%d]", gzc.conn.SessionID())
}

// 该方法用于集群服务的配置项统一管理，client监听该节点的Data数据的变化
// servicepath 要创建的临时节点路径
// data 为该节点数据(配置数据)
// createFatherNodePaths 表示如果rootPath这个路径上存在还未建立的一层父节点，是否默认创建该父节点
func (gzs *GozkServer) ServiceConfig(servicepath string, data []byte, createFatherNodePaths bool) error {
	servicepath = path.Clean(servicepath)
	err := gzs.checkRoot(servicepath, createFatherNodePaths)
	if err != nil {
		return err
	}

	_, err = gzs.conn.Create(servicepath, data, gozk.FlagPersistent, gozk.WorldACL(gozk.PermAll))
	if err != nil {
		return err
	}

	return nil
}

// 该方法用于服务注册和发现的场景，服务器端注册zookeeper服务节点
// rootPath 为父节点，client监听该节点下Children节点的变化
// serviceHost 要创建的临时节点(建议以该服务的IP:port来命名该节点)
// data 为该节点数据
// createFatherNodePaths 表示如果rootPath这个路径上存在还未建立的一层父节点，是否默认创建该父节点
func (gzs *GozkServer) ServiceRegistry(rootPath, serviceHost string, data []byte, createFatherNodePaths bool) error {
	abpath := path.Join(rootPath, serviceHost)
	err := gzs.checkRoot(abpath, createFatherNodePaths)
	if err != nil {
		return err
	}

	_, err = gzs.conn.Create(abpath, data, gozk.FlagEphemeral, gozk.WorldACL(gozk.PermAll))
	if err != nil {
		return err
	}

	return nil
}

// 检测父节点是否存在，如不存在，且createRoots == true 则创建各级父节点
func (gzs *GozkServer) checkRoot(path string, createFatherNodePaths bool) error {
	rootPaths := getFatherNodePaths(path)
	for _, v := range rootPaths {
		exist, _, err := gzs.conn.Exists(v)
		if err != nil {
			return err
		}

		if !exist {
			if !createFatherNodePaths {
				return fmt.Errorf("%s can not find father node %s", gzs, v)
			}

			_, err = gzs.conn.Create(v, s2b(v), gozk.FlagPersistent, gozk.WorldACL(gozk.PermAll))
			if err != nil {
				return err
			}
		}

		log.Debugf("rootPath %s is ready", v)
	}
	return nil
}

func getFatherNodePaths(path string) []string {
	path = strings.Trim(path, "/")
	path = "/" + path

	ab_paths := strings.Split(path, "/")
	count := len(ab_paths)

	for i := 0; i < count-1; i++ {
		ab_paths[i+1] = strings.Join(ab_paths[i:i+2], "/")
	}

	return ab_paths[1:count-1]
}

func s2b(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}
