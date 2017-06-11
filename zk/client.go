package zk

import (
	"fmt"
	"sync"
	"time"

	"strings"
	"git.oschina.net/liuxp/common.git/log"
	"github.com/samuel/go-zookeeper/zk"
)

type GozkClient struct {
	path     string
	conn     *zk.Conn
	data     []byte
	children []string

	lock *sync.RWMutex
}

func NewGozkClient(zkhosts []string, nodepath string) (*GozkClient, error) {
	nodepath = strings.Trim(nodepath, "/")
	nodepath = "/" + nodepath

	client := &GozkClient{
		path: nodepath,
		lock: &sync.RWMutex{},
	}

	c, _, err := zk.Connect(zkhosts, 2*time.Second)

	client.conn = c
	data, _, err := c.Get(nodepath)
	if err != nil {
		return nil, err
	}

	client.data = data

	children, _, err := c.Children(nodepath)
	if err != nil {
		return nil, err
	}

	client.children = children

	go client.watchNodeDataChanged()
	go client.watchNodeChildrenChanged()

	return client, nil
}

func (gzc *GozkClient) String() string {
	return fmt.Sprintf("go-zookeeper Client sid[%d] path[%s]", gzc.conn.SessionID(), gzc.path)
}

func (gzc *GozkClient) GetData() []byte {
	gzc.lock.RLock()
	defer gzc.lock.RUnlock()
	return gzc.data
}

func (gzc *GozkClient) GetChildren() []string {
	gzc.lock.RLock()
	defer gzc.lock.RUnlock()
	return gzc.children
}

func (gzc *GozkClient) watchNodeDataChanged() {
	for {
		_, _, events, err := gzc.conn.GetW(gzc.path)
		if err != nil {
			log.Error(err)
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

			gzc.lock.Lock()
			gzc.data = data
			gzc.lock.Unlock()
		}
	}
}

func (gzc *GozkClient) watchNodeChildrenChanged() {
	for {
		_, _, events, err := gzc.conn.ChildrenW(gzc.path)
		if err != nil {
			log.Error(err)
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
			gzc.lock.Lock()
			gzc.children = children
			gzc.lock.Unlock()
		}
	}
}
