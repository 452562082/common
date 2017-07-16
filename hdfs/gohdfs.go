package hdfs

import (
	"fmt"
	"git.oschina.net/kuaishangtong/common/utils/httplib"
	"git.oschina.net/kuaishangtong/common/utils/log"
	"github.com/colinmarc/hdfs"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

type HdfsClient struct {
	conn_hdfs_addr string
	client         *hdfs.Client
	host_web_map   map[string]string
	web_host_map   map[string]string
}

var defaultHdfsClient *HdfsClient

type NameNodeStatus struct {
	Beans []struct {
		State string `json:"State"`
	} `json:"beans"`
}

func Init(hdfs_addr, hdfs_http_addr []string) error {
	var err error

	defaultHdfsClient, err = NewHdfsClient(hdfs_addr, hdfs_http_addr)
	if err != nil {
		return err

	}
	return nil
}

func CheckHDFSAlive(hdfs_web_host string) (bool, error) {
	var nnStatus NameNodeStatus
	err := httplib.Get(hdfs_web_host + "/jmx?qry=Hadoop:service=NameNode,name=NameNodeStatus").ToJson(&nnStatus)
	if err != nil {
		return false, err
	}

	if len(nnStatus.Beans) < 1 {
		return false, fmt.Errorf("bean in NameNodeStatus is nil")
	}

	state := nnStatus.Beans[0].State

	return state == "active", nil
}

func NewHdfsClient(hdfs_addrs, hdfs_http_addrs []string) (*HdfsClient, error) {

	if len(hdfs_addrs) == 0 {
		return nil, fmt.Errorf("hdfs host array length can not be 0")
	}

	if len(hdfs_addrs) != len(hdfs_http_addrs) {
		return nil, fmt.Errorf("hdfs host array length must be equal to hdfs http address array lentgh")
	}

	var host_web_map map[string]string = make(map[string]string)
	var web_host_map map[string]string = make(map[string]string)

	for _, http_addr := range hdfs_http_addrs {
		for _, addr := range hdfs_addrs {
			if strings.Split(http_addr, ":")[0] == strings.Split(addr, ":")[0] {
				host_web_map[addr] = http_addr
				web_host_map[http_addr] = addr
			}
		}
	}

	log.Info(host_web_map)

	var err error
	var client *hdfs.Client
	var addr string

	for _, addr = range hdfs_addrs {
		client, err = hdfs.New(addr)
		if err != nil {
			log.Errorf("hdfs connect to %s failed: %v", addr, err)
			continue
		}
		goto END
	}

END:
	if err != nil {
		return nil, err
	}

	hclient := &HdfsClient{
		conn_hdfs_addr: addr,
		client:         client,
		host_web_map:   host_web_map,
		web_host_map:   web_host_map,
	}

	go hclient.checkLoop(hdfs_http_addrs)

	return hclient, nil
}

func (hc *HdfsClient) checkLoop(hdfs_addrs []string) {
	var alive map[string]struct{} = make(map[string]struct{})

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, v := range hdfs_addrs {
				active, err := CheckHDFSAlive(v)
				if err != nil {
					log.Error(err)
					goto NEXT
				}

				if active {
					alive[v] = struct{}{}
				} else {
					delete(alive, v)
				}

			NEXT:
			}

			if _, ok := alive[hc.host_web_map[hc.conn_hdfs_addr]]; !ok && len(alive) > 0 {
				for http_addr, _ := range alive {
					err := hc.ResetHDFSConnection(hc.web_host_map[http_addr])
					if err != nil {
						log.Error(err)
						continue
					}
					log.Infof("Reconnect to HDFS %s success", hc.web_host_map[http_addr])
					goto END
				}
			}
		END:
		}
	}
}

func (hc *HdfsClient) ResetHDFSConnection(hdfsaddr string) error {
	var err error

	hc.client.Close()

	hc.client, err = hdfs.New(hdfsaddr)
	if err != nil {
		return err
	}

	return nil
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

func (hc *HdfsClient) CopyFileToRemote(local_file_path, hdfs_file_path string) error {
	err := checkPath(hdfs_file_path)
	if err != nil {
		return err
	}

	err = hc.MkdirAll(path.Dir(hdfs_file_path), 0755)
	if err != nil {
		return err
	}

	return hc.client.CopyToRemote(local_file_path, hdfs_file_path)
}

func (hc *HdfsClient) MkdirAll(dir string, perm os.FileMode) error {
	return hc.client.MkdirAll(dir, perm)
}

func GetWaveFromHDFS(hdfsfile string) ([]byte, error) {
	return defaultHdfsClient.ReadFile(hdfsfile)
}

func CopyModelFromHDFS(hdfsfile string) error {
	return defaultHdfsClient.CopyFileToLocal(hdfsfile, hdfsfile)
}

func (hc *HdfsClient) ReadFile(filename string) ([]byte, error) {
	err := checkPath(filename)
	if err != nil {
		return nil, err
	}
	return hc.client.ReadFile(filename)
}

func (hc *HdfsClient) ReadDir(filename string) ([]os.FileInfo, error) {
	err := checkPath(filename)
	if err != nil {
		return nil, err
	}
	return hc.client.ReadDir(filename)
}

func (hc *HdfsClient) CopyFileToLocal(hdfs_file_path, local_file_path string) error {
	err := checkPath(hdfs_file_path)
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Dir(local_file_path), 0755)
	if err != nil {
		return err
	}

	return hc.client.CopyToLocal(hdfs_file_path, local_file_path)
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

	err = os.MkdirAll(path.Dir(localdir), 0755)
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

func InitSyncModel(hdfs_addr, hdfs_http_addr []string, modeldir string) error {

	if modeldir[len(modeldir)-1] != '/' {
		modeldir += "/"
	}
	hdfsClient, err := NewHdfsClient(hdfs_addr, hdfs_http_addr)
	if err != nil {
		return err
	}

	var localmap, hdfsmap map[string]struct{} = make(map[string]struct{}), make(map[string]struct{})

	local_file_infos, err := ioutil.ReadDir(modeldir)
	if err != nil {
		return err
	}

	for i, v := range local_file_infos {
		if i < 5 {
			log.Debug(modeldir + v.Name())
		}
		localmap[modeldir+v.Name()] = struct{}{}
	}

	log.Infof("catch local ivfiles, count: %d", len(local_file_infos))

	hdfs_file_infos, err := hdfsClient.ReadDir(modeldir)
	if err != nil {
		return err
	}

	for i, v := range hdfs_file_infos {
		if i < 5 {
			log.Debug(modeldir + v.Name())
		}
		hdfsmap[modeldir+v.Name()] = struct{}{}
	}

	log.Infof("catch hdfs ivfiles, count: %d", len(hdfs_file_infos))

	for k, _ := range hdfsmap {

		if _, ok := localmap[k]; ok {
			delete(localmap, k)
		}

		delete(hdfsmap, k)
	}

	download := len(hdfsmap)

	log.Infof("%d need to download to local", download)

	delete := len(localmap)

	log.Infof("%d need to delete", delete)

	for k, _ := range localmap {
		err := os.Remove(k)
		if err != nil {
			log.Error(err)
		}
		log.Debugf("remove %s", k)
	}

	for k, _ := range hdfsmap {
		err := hdfsClient.CopyFileToLocal(k, k)
		if err != nil {
			log.Error(err)
		}

		log.Debugf("download %s", k)
	}

	err = hdfsClient.Close()
	if err != nil {
		return err
	}

	return nil
}
