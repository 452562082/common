// Example: index a document, search for it, then delete it.
//
//	ES_HOSTS=http://localhost:9200 go run ./examples/elastic_basic
package main

import (
	"context"
	"fmt"
	"log"

	"common/elastic"
	"common/env"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	addrs, err := env.ElasticHosts()
	if err != nil {
		log.Fatalf("ES_HOSTS unset: %v", err)
	}

	cli, err := elastic.NewClient(elastic.Config{Addresses: addrs})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx := context.Background()

	if err := cli.IndexWithRefresh(ctx, "users", "u-1", User{Name: "Ada", Age: 36}, elastic.RefreshWaitFor); err != nil {
		log.Fatalf("index: %v", err)
	}

	res, err := cli.Search(ctx, []string{"users"}, map[string]any{
		"query": map[string]any{"match": map[string]any{"name": "ada"}},
	})
	if err != nil {
		log.Fatalf("search: %v", err)
	}
	fmt.Printf("hits: %d\n", res.TotalHits)

	if err := cli.Delete(ctx, "users", "u-1"); err != nil {
		log.Fatalf("delete: %v", err)
	}
}
