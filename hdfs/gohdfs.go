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
	defaultHdfsClient, err := NewHdfsClient(address)
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
	if !strings.HasPrefix(filename, "/") {
		return fmt.Errorf("filename must be a absolute path")
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
	return defaultHdfsClient.ReadFile(filename)
}

func (hc *HdfsClient) ReadFile(filename string) ([]byte, error) {
	return hc.client.ReadFile(filename)
}

func (hc *HdfsClient) Append(filename string, data []byte) error {
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
	return hc.client.Remove(filename)
}

func (hc *HdfsClient) Close() error {
	return hc.client.Close()
}
