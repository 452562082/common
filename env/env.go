package env

import (
	"fmt"
	"os"
	"strings"
)

const (
	__ZK_HOSTS    = "ZK_HOSTS"
	__KAFKA_HOSTS = "KAFKA_HOSTS"
	__HDFS_HOSTS  = "HDFS_HOSTS"
	__ES_HOSTS    = "ES_HOSTS"
	__HOST_IP     = "HOST_IP"
	__SERVER_HOST = "SERVER_HOST"
)

var zk_hosts string
var kafka_hosts string
var hdfs_hosts string
var es_hosts string
var host_ip string
var server_host string

//func init() {
//	zk_hosts = os.Getenv(__ZK_HOSTS)
//	kafka_hosts = os.Getenv(__KAFKA_HOSTS)
//	hdfs_hosts = os.Getenv(__HDFS_HOSTS)
//	es_hosts = os.Getenv(__ES_HOSTS)
//}

func GetServerHost() (string, error) {
	var isSet bool
	server_host, isSet = os.LookupEnv(__SERVER_HOST)
	if !isSet {
		return "", fmt.Errorf("env %s is an unset value", __SERVER_HOST)
	}
	return host_ip, nil
}

func GetHostIp() (string, error) {
	var isSet bool
	host_ip, isSet = os.LookupEnv(__HOST_IP)
	if !isSet {
		return "", fmt.Errorf("env %s is an unset value", __HOST_IP)
	}
	return host_ip, nil
}

func GetZookeeperHosts() ([]string, error) {
	var isSet bool
	zk_hosts, isSet = os.LookupEnv(__ZK_HOSTS)
	if !isSet {
		return nil, fmt.Errorf("env %s is an unset value", __ZK_HOSTS)
	}

	if len(zk_hosts) > 0 {
		return strings.Split(zk_hosts, ";"), nil
	}
	return nil, fmt.Errorf("value of env %s is \"\"", __ZK_HOSTS)
}

func GetKafkaHosts() ([]string, error) {
	var isSet bool
	kafka_hosts, isSet = os.LookupEnv(__KAFKA_HOSTS)
	if !isSet {
		return nil, fmt.Errorf("env %s is an unset value", __KAFKA_HOSTS)
	}

	if len(kafka_hosts) > 0 {
		return strings.Split(kafka_hosts, ";"), nil
	}
	return nil, fmt.Errorf("value of env %s is \"\"", __KAFKA_HOSTS)
}

func GetHDFSHosts() ([]string, error) {
	var isSet bool
	hdfs_hosts, isSet = os.LookupEnv(__HDFS_HOSTS)
	if !isSet {
		return nil, fmt.Errorf("env %s is an unset value", __HDFS_HOSTS)
	}

	if len(hdfs_hosts) > 0 {
		return strings.Split(hdfs_hosts, ";"), nil
	}

	return nil, fmt.Errorf("value of env %s is \"\"", __HDFS_HOSTS)
}

func GetElasticSearchHosts() ([]string, error) {
	var isSet bool
	es_hosts, isSet = os.LookupEnv(__ES_HOSTS)
	if !isSet {
		return nil, fmt.Errorf("env %s is an unset value", __ES_HOSTS)
	}
	if len(es_hosts) > 0 {
		return strings.Split(es_hosts, ";"), nil
	}
	return nil, fmt.Errorf("value of env %s is \"\"", __ES_HOSTS)
}
