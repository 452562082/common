package hdfs

import (
	"fmt"
	"github.com/colinmarc/hdfs"
	"strings"
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

	fInfos, err := hc.client.ReadDir(hdfsdir)
	if err != nil {
		return err
	}

	for _, fi := range fInfos {
		err = hc.client.CopyToLocal(hdfsdir+"/"+fi.Name(), localdir+"/"+fi.Name())
		if err != nil {
			return err
		}
	}

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
