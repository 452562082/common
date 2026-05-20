// Example: ephemeral service registration. The node disappears when the
// process exits (or its ZK session expires).
//
//	ZK_HOSTS=localhost:2181 go run ./examples/zk_register
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"common/env"
	"common/zk"
)

func main() {
	hosts, err := env.ZooKeeperHosts()
	if err != nil {
		log.Fatalf("ZK_HOSTS unset: %v", err)
	}

	srv, err := zk.NewServer(hosts)
	if err != nil {
		log.Fatalf("new zk server: %v", err)
	}
	defer func() { _ = srv.Close() }()

	host, _ := os.Hostname()
	if err := srv.Register("/services/demo", host, []byte("ready"), true); err != nil {
		log.Fatalf("register: %v", err)
	}
	log.Printf("registered /services/demo/%s — press Ctrl-C to exit", host)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
