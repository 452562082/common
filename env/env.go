// Package env reads infrastructure endpoints from delimited env vars.
//
// Host lists accept both comma- and semicolon-separated values. All getters
// return ErrUnset (wrapped) when the variable is missing or empty, so callers
// can distinguish "use a default" from "configured but blank".
package env

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	envZKHosts    = "ZK_HOSTS"
	envKafkaHosts = "KAFKA_HOSTS"
	envHDFSHosts  = "HDFS_HOSTS"
	envESHosts    = "ES_HOSTS"
	envRedisHosts = "REDIS_HOSTS"
	envHostIP     = "HOST_IP"
	envServerHost = "SERVER_HOST"
)

// ErrUnset is returned (wrapped) when an env var is missing or empty.
var ErrUnset = errors.New("env var is unset")

// splitHosts splits on comma or semicolon and trims each element.
// Empty elements are dropped.
func splitHosts(v string) []string {
	fields := strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ';' })
	out := fields[:0]
	for _, f := range fields {
		if s := strings.TrimSpace(f); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func getHosts(name string) ([]string, error) {
	v, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(v) == "" {
		return nil, fmt.Errorf("%s: %w", name, ErrUnset)
	}
	hosts := splitHosts(v)
	if len(hosts) == 0 {
		return nil, fmt.Errorf("%s: %w", name, ErrUnset)
	}
	return hosts, nil
}

func getString(name string) (string, error) {
	v, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("%s: %w", name, ErrUnset)
	}
	return v, nil
}

func ServerHost() (string, error)       { return getString(envServerHost) }
func HostIP() (string, error)           { return getString(envHostIP) }
func ZooKeeperHosts() ([]string, error) { return getHosts(envZKHosts) }
func KafkaHosts() ([]string, error)     { return getHosts(envKafkaHosts) }
func HDFSHosts() ([]string, error)      { return getHosts(envHDFSHosts) }
func ElasticHosts() ([]string, error)   { return getHosts(envESHosts) }
func RedisHosts() ([]string, error)     { return getHosts(envRedisHosts) }
