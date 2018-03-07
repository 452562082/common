package main

import (
	"flag"
	"fmt"
	"kuaishangtong/common/elastic"
	"kuaishangtong/common/utils/log"
	"strings"
)

var hosts string
var srcindex string
var dstindex string

func init() {
	flag.StringVar(&hosts, "hosts", "", "elasticsearch cluster hosts")
	flag.StringVar(&srcindex, "srcindex", "", "src es index")
	flag.StringVar(&dstindex, "dstindex", "", "dst es index")
}

func main() {

	flag.Parse()
	if len(hosts) == 0 {
		log.Fatal("hosts can not be \"\"")
	}
	if len(srcindex) == 0 {
		log.Fatal("srcindex can not be \"\"")
	}
	if len(dstindex) == 0 {
		log.Fatal("dstindex can not be \"\"")
	}

	eshosts := strings.Split(hosts, ";")

	__elastic_client_src, err := elastic.NewElasticClient(eshosts)
	if err != nil {
		log.Fatal(err)
	}

	__elastic_client_dst, err := elastic.NewElasticClient(eshosts)
	if err != nil {
		log.Fatal(err)
	}

	result, err := __elastic_client_src.WildcardQuery(srcindex, srcindex, "vpr_utt_node", "*")
	if err != nil {
		log.Fatal(err)
	}

	if result.Hits == nil {
		log.Fatalf("wildcard query in src index %s failed", srcindex)
	}

	exist, err := __elastic_client_dst.IndexExists(dstindex)
	if err != nil {
		log.Fatal(err)
	}

	if !exist {
		_, err = __elastic_client_dst.CreateIndexBodyString(dstindex, fmt.Sprintf(elastic.ASV_VPR_INFO_INDEX, dstindex))
		if err != nil {
			log.Fatalf("CreateIndexBodyString error: %v", err)
		}
		log.Infof("__elastic_client create indexBodyString %s", dstindex)
	}

	finish := 0
	if result.Hits.TotalHits > 0 {
		log.Infof("data from src index, total: %d", result.Hits.TotalHits)

		for _, hit := range result.Hits.Hits {
			insertresult, err := __elastic_client_dst.InsertDocBodyStringWithID(dstindex, dstindex, hit.Id, string(*hit.Source))
			if err != nil {
				log.Fatalf("insert data %s to dst index %s failed", string(*hit.Source), dstindex)
			}
			if insertresult.Created {
				finish++
				if finish%10 == 0 {
					log.Infof("transfer finish %d", finish)
				}
			}
		}
	}
	log.Infof("transfer done success %d", finish)
}
