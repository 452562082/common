package hdfs

import (
	"fmt"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"github.com/colinmarc/hdfs"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

type HdfsClient struct {
	client *hdfs.Client
}

var defaultHdfsClient *HdfsClient

func Init(address string) error {
	var err error
	defaultHdfsClient, err = NewHdfsClient(address)
	if err != nil {
		return err
	}
	return nil
}

func NewHdfsClient(address string) (*HdfsClient, error) {
	client, err := hdfs.New(address)
	if err != nil {
		return nil, err
	}

	return &HdfsClient{
		client: client,
	}, nil
}

func (hc *HdfsClient) WriteFile(filename string, data []byte) error {
	err := checkPath(filename)
	if err != nil {
		return err
	}

	err = hc.MkdirAll(path.Dir(filename), 0755)
	if err != nil {
		return err
	}

	fw, err := hc.client.Create(filename)
	if err != nil {
		return err
	}
	defer fw.Close()

	_, err = fw.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (hc *HdfsClient) MkdirAll(dir string, perm os.FileMode) error {
	return hc.client.MkdirAll(dir, perm)
}

func GetWaveFromHDFS(hdfsfile string) ([]byte, error) {
	return defaultHdfsClient.ReadFile(hdfsfile)
}

func GetWaveFromHDFS2(hdfsaddr, hdfsfile string) ([]byte, error) {
	client, err := NewHdfsClient(hdfsaddr)
	if err != nil {
		return nil, err
	}

	return client.ReadFile(hdfsfile)
}

func (hc *HdfsClient) ReadFile(filename string) ([]byte, error) {
	err := checkPath(filename)
	if err != nil {
		return nil, err
	}
	return hc.client.ReadFile(filename)
}

func (hc *HdfsClient) CopyAllFilesToLocal(hdfsdir, localdir string) error {

	if localdir[len(localdir)-1] != '/' {
		localdir += "/"
	}

	if hdfsdir[len(hdfsdir)-1] != '/' {
		hdfsdir += "/"
	}

	fInfos, err := hc.client.ReadDir(hdfsdir)
	if err != nil {
		return err
	}

	exit := make(chan bool)

	index := 0
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Infof("downloading count: %d", index)
			case <-exit:
				log.Infof("download done total: %d", index)
				return
			}
		}
	}()

	for _, fi := range fInfos {
		err = hc.client.CopyToLocal(hdfsdir+fi.Name(), localdir+fi.Name())
		if err != nil {
			return err
		}
		index++
	}

	close(exit)
	time.Sleep(1 * time.Second)

	return nil
}
func (hc *HdfsClient) CopyAllFilesToRemote(localdir, hdfsdir string) error {

	if localdir[len(localdir)-1] != '/' {
		localdir += "/"
	}

	if hdfsdir[len(hdfsdir)-1] != '/' {
		hdfsdir += "/"
	}

	fInfos, err := ioutil.ReadDir(localdir)
	if err != nil {
		return err
	}

	err = hc.MkdirAll(hdfsdir, 0755)
	if err != nil {
		return err
	}

	exit := make(chan bool)

	index := 0
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Infof("uploading count: %d", index)
			case <-exit:
				log.Infof("upload done total: %d", index)
				return
			}
		}
	}()

	for _, fi := range fInfos {
		err = hc.client.CopyToRemote(localdir+fi.Name(), hdfsdir+fi.Name())
		if err != nil {
			return err
		}

		index++
	}

	close(exit)
	time.Sleep(1 * time.Second)

	return nil
}

func (hc *HdfsClient) Append(filename string, data []byte) error {
	err := checkPath(filename)
	if err != nil {
		return err
	}

	fw, err := hc.client.Append(filename)
	if err != nil {
		return nil
	}
	defer fw.Close()

	_, err = fw.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (hc *HdfsClient) Remove(filename string) error {
	err := checkPath(filename)
	if err != nil {
		return err
	}
	return hc.client.Remove(filename)
}

func (hc *HdfsClient) Close() error {
	return hc.client.Close()
}

func checkPath(path string) error {
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("filename must be a absolute path")
	}
	return nil
}
