package zk

import (
	"fmt"
	"strings"
	"time"

	"git.oschina.net/kuaishangtong/common/utils/log"
	"github.com/samuel/go-zookeeper/zk"
)

type GozkClient struct {
	path     string
	conn     *zk.Conn
	data     chan []byte
	children chan []string
}

func NewGozkClient(zkhosts []string, nodepath string, _default []byte) (*GozkClient, error) {
	nodepath = strings.Trim(nodepath, "/")
	nodepath = "/" + nodepath

	client := &GozkClient{
		path:     nodepath,
		data:     make(chan []byte),
		children: make(chan []string),
	}

	c, _, err := zk.Connect(zkhosts, 2*time.Second)
	if err != nil {
		return nil, err
	}

	client.conn = c
	exist, _, err := client.conn.Exists(nodepath)
	if !exist {
		_, err = client.conn.Create(nodepath, _default, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			return nil, err
		}
	}

	go client.watchNodeDataChanged()
	go client.watchNodeChildrenChanged()

	return client, nil
}

func (gzc *GozkClient) String() string {
	return fmt.Sprintf("go-zk Client sid[%d] path[%s]", gzc.conn.SessionID(), gzc.path)
}

func (gzc *GozkClient) GetData() <-chan []byte {
	return gzc.data
}

func (gzc *GozkClient) GetChildren() <-chan []string {
	return gzc.children
}

func (gzc *GozkClient) GetChildrenOnce(node string) ([]string, *zk.Stat, error) {
	return gzc.conn.Children(node)
}

func (gzc *GozkClient) Close() {
	gzc.conn.Close()
}

func (gzc *GozkClient) watchNodeDataChanged() {
	first := true
	for {
		data, _, events, err := gzc.conn.GetW(gzc.path)
		if err != nil {
			log.Error(err)
			continue
		}

		if first {
			gzc.data <- data
			first = false
			continue
		}

		evt := <-events
		if evt.Err != nil {
			log.Error(evt.Err)
			continue
		}

		if evt.Type == zk.EventNodeDataChanged {
			data, _, err := gzc.conn.Get(gzc.path)
			if err != nil {
				log.Error(err)
				continue
			}

			gzc.data <- data
		}
	}
}

func (gzc *GozkClient) watchNodeChildrenChanged() {
	first := true
	for {
		children, _, events, err := gzc.conn.ChildrenW(gzc.path)
		if err != nil {
			log.Error(err)
			continue
		}

		if first {
			gzc.children <- children
			first = false
			continue
		}

		evt := <-events
		if evt.Err != nil {
			log.Error(evt.Err)
			continue
		}

		if evt.Type == zk.EventNodeChildrenChanged {
			children, _, err := gzc.conn.Children(gzc.path)
			if err != nil {
				log.Error(err)
				continue
			}

			gzc.children <- children
		}
	}
}
