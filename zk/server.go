package zk

import (
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"git.oschina.net/kuaishangtong/common/utils"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"github.com/samuel/go-zookeeper/zk"
)

var SessionTimeout int = 2000

type createType int32

func (t createType) String() string {
	if name := createTypeName[t]; name != "" {
		return name
	}
	return "Unknown"
}

const (
	_ZK_CREATE_NODE_DATA     createType = 0
	_ZK_CREATE_NODE_CHILDREN createType = 1
)

var (
	createTypeName = map[createType]string{
		_ZK_CREATE_NODE_DATA:     "ZK_CREATE_NODE_DATA",
		_ZK_CREATE_NODE_CHILDREN: "ZK_CREATE_NODE_CHILDREN",
	}
)

type zknode struct {
	typ    createType // 0: for data, 1: for children
	path   string
	data   []byte
	active bool
}

type GozkServer struct {
	conn        *zk.Conn
	connEvent   <-chan zk.Event
	registryMap map[string]*zknode
	lock        *sync.RWMutex
}

func NewGozkServer(zkhosts []string) (*GozkServer, error) {
	conn, event, err := zk.Connect(zkhosts, time.Duration(SessionTimeout)*time.Millisecond)
	if err != nil {
		return nil, err
	}

	gzs := &GozkServer{
		conn:        conn,
		connEvent:   event,
		registryMap: make(map[string]*zknode),
		lock:        &sync.RWMutex{},
	}
	go gzs.loop()

	return gzs, nil
}

func (gzs *GozkServer) String() string {
	return fmt.Sprintf("go-zk Server sid[%d]", gzs.conn.SessionID())
}

func (gzs *GozkServer) Close() {
	gzs.conn.Close()
}

func (gzs *GozkServer) loop() {
	for {
		select {
		case evt, ok := <-gzs.connEvent:
			if ok && evt.Type == zk.EventSession {
				switch evt.State {
				case zk.StateHasSession:
					log.Infof("zk conn %s has session", gzs.conn.Server())
					gzs.lock.Lock()
					for path, node := range gzs.registryMap {
						if !node.active {
							switch node.typ {
							case _ZK_CREATE_NODE_DATA:
								err := gzs.serviceConfig(path, node.data, true)
								if err != nil {
									log.Error(err)
									continue
								}
								log.Infof("zk conn %s recreate node in path %s", gzs.conn.Server(), path)
							case _ZK_CREATE_NODE_CHILDREN:
								parts := strings.Split(path, "/")
								err := gzs.serviceRegistry(strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1], node.data, true)
								if err != nil {
									log.Error(err)
									continue
								}
								log.Infof("zk conn %s recreate node in path %s", gzs.conn.Server(), path)
							}
						}
					}
					gzs.lock.Unlock()

				case zk.StateDisconnected:
					log.Warnf("zk conn %s disconnect", gzs.conn.Server())
					gzs.lock.Lock()
					for _, node := range gzs.registryMap {
						node.active = false
					}
					gzs.lock.Unlock()

				case zk.StateConnected:
					log.Infof("zk conn %s connected", gzs.conn.Server())

				case zk.StateExpired:
					log.Warnf("zk conn %s expired", gzs.conn.Server())
				}
			}
		}
	}
}

// 该方法用于集群服务的配置项统一管理，client监听该节点的Data数据的变化
// servicepath 要创建的临时节点路径
// data 为该节点数据(配置数据)
// createFatherNodePaths 表示如果rootPath这个路径上存在还未建立的一层父节点，是否默认创建该父节点
func (gzs *GozkServer) ServiceConfig(servicepath string, data []byte, createFatherNodePaths bool) error {
	gzs.lock.Lock()
	defer gzs.lock.Unlock()
	return gzs.serviceConfig(servicepath, data, createFatherNodePaths)
}

func (gzs *GozkServer) serviceConfig(servicepath string, data []byte, createFatherNodePaths bool) error {
	servicepath = path.Clean(servicepath)
	err := gzs.checkRoot(servicepath, createFatherNodePaths)
	if err != nil {
		return err
	}

	node, ok := gzs.registryMap[servicepath]
	if ok {
		if node.active {
			return nil
		}
	} else {
		node = &zknode{
			typ:    _ZK_CREATE_NODE_DATA,
			path:   servicepath,
			data:   data,
			active: false,
		}
		gzs.registryMap[servicepath] = node
	}

	_, err = gzs.conn.Create(servicepath, data, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		if err == zk.ErrNodeExists { // 如果要创建的节点已经存在，直接修改该节点的数据
			_, err = gzs.conn.Set(servicepath, data, -1)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	gzs.registryMap[servicepath].active = true

	return nil
}

// 该方法用于服务注册和发现的场景，服务器端注册zookeeper服务节点
// rootPath 为父节点，client监听该节点下Children节点的变化
// serviceHost 要创建的临时节点(建议以该服务的IP:port来命名该节点)
// data 为该节点数据
// createFatherNodePaths 表示如果rootPath这个路径上存在还未建立的一层父节点，是否默认创建该父节点
func (gzs *GozkServer) ServiceRegistry(rootPath, serviceHost string, data []byte, createFatherNodePaths bool) error {
	gzs.lock.Lock()
	defer gzs.lock.Unlock()
	return gzs.serviceRegistry(rootPath, serviceHost, data, createFatherNodePaths)
}

func (gzs *GozkServer) serviceRegistry(rootPath, serviceHost string, data []byte, createFatherNodePaths bool) error {
	abpath := path.Join(rootPath, serviceHost)
	err := gzs.checkRoot(abpath, createFatherNodePaths)
	if err != nil {
		return err
	}

	node, ok := gzs.registryMap[abpath]
	if ok {
		if node.active {
			return nil
		}
	} else {
		node = &zknode{
			typ:    _ZK_CREATE_NODE_CHILDREN,
			path:   abpath,
			data:   data,
			active: false,
		}
		gzs.registryMap[abpath] = node
	}

	_, err = gzs.conn.Create(abpath, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		return err
	}

	gzs.registryMap[abpath].active = true

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

			log.Debug("create father:", v)
			_, err = gzs.conn.Create(v, utils.S2B(v), 0, zk.WorldACL(zk.PermAll))
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

	return ab_paths[1 : count-1]
}
