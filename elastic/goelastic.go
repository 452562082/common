package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"kuaishangtong/asvWebApi/const"
	"kuaishangtong/common/hdfs"
	"kuaishangtong/common/utils"
	"gopkg.in/olivere/elastic.v2"
	"os"
	"strconv"
)

var ASV_VPR_INFO_INDEX string = `{
	  "settings": {
		"analysis": {
		  "analyzer": {
			"my_analyzer": {
			  "tokenizer": "my_tokenizer"
			}
		  },
		  "tokenizer": {
			"my_tokenizer": {
			  "type": "ngram",
			  "min_gram": 1,
			  "max_gram": 20,
			  "token_chars": [
				"letter",
            	"digit",
				"punctuation",
				"symbol"
			  ]
			}
		  }
		}
	  },

		"mappings": {
		  "%s": {
			"properties": {
			  "vpr_task_id": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_spk_id": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_wav_file": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_utt_node": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_utt_recordid": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_add_time": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "date"
			  },
			  "vpr_utt_duration": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_utt_valid_dura": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_utt_chan": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_utt_dir": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_utt_gender": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_norm_params": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_utt_scene": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  },
			  "vpr_has_tone": {
				"index": "not_analyzed",
				"store": "yes",
				"type": "string"
			  }
			}
		  }
		}
	}
`

type ElasticClient struct {
	client *elastic.Client
}

func NewElasticClient(hosts []string) (*ElasticClient, error) {
	urls := make([]string, 0)
	for i := 0; i < len(hosts); i++ {
		_url := fmt.Sprintf("http://%s", hosts[i])
		urls = append(urls, _url)
	}

	client, err := elastic.NewClient(elastic.SetURL(urls...), elastic.SetErrorLog(new(utils.ErrLog)), elastic.SetInfoLog(new(utils.InfoLog)))
	if err != nil {
		return nil, err
	}

	__elastic_client := &ElasticClient{
		client: client,
	}

	return __elastic_client, nil
}

func (ec *ElasticClient) IndexExists(indices string) (bool, error) {
	return ec.client.IndexExists(indices).Do()
}

func (ec *ElasticClient) CreateIndexBodyString(name string, body string) (*elastic.CreateIndexResult, error) {
	return ec.client.CreateIndex(name).Body(body).Do()
}

//func (ec *ElasticClient) CreateIndex(name string) (*elastic.CreateIndexResult, error) {
//	return ec.client.CreateIndex(name).Do()
//}
//
//func (ec *ElasticClient) CreateIndexBodyJson(name string, body interface{}) (*elastic.CreateIndexResult, error) {
//	return ec.client.CreateIndex(name).BodyJson(body).Do()
//}

func (ec *ElasticClient) DeleteIndex(indices string) (*elastic.DeleteIndexResult, error) {
	return ec.client.DeleteIndex(indices).Do()
}

func (ec *ElasticClient) DeleteDocWithID(index, typ, id string) (*elastic.DeleteResult, error) {
	return ec.client.Delete().Index(index).Type(typ).Id(id).Do()
}

//func (ec *ElasticClient) DeleteDoc(index, typ string) (*elastic.DeleteResult, error) {
//	return ec.client.Delete().Index(index).Type(typ).Do()
//}

func (ec *ElasticClient) UpdateDocBodyWithID(index, typ, id string, data map[string]interface{}) error {
	// Update (non-existent) tweet with id #1
	_, err := ec.client.Update().
		Index(index).Type(typ).Fields().
		Doc(data).
		Do()

	if err != nil {
		return err
	}
	return nil
}

func (ec *ElasticClient) InsertDocBodyJsonWithID(index, typ, id string, body interface{}) (*elastic.IndexResult, error) {
	return ec.client.Index().Index(index).Type(typ).Id(id).BodyJson(body).Do()
}

func (ec *ElasticClient) InsertDocBodyStringWithID(index, typ, id string, body string) (*elastic.IndexResult, error) {
	return ec.client.Index().Index(index).Type(typ).Id(id).BodyString(body).Do()
}

func (ec *ElasticClient) InsertDocBodyJson(index, typ string, body interface{}) (*elastic.IndexResult, error) {
	return ec.client.Index().Index(index).Type(typ).BodyJson(body).Do()
}

func (ec *ElasticClient) InsertDocBodyString(index, typ string, body string) (*elastic.IndexResult, error) {
	return ec.client.Index().Index(index).Type(typ).BodyString(body).Do()
}

func (ec *ElasticClient) GetDoc(index, typ, id string) (*elastic.GetResult, error) {
	return ec.client.Get().Index(index).Type(typ).Id(id).Do()
}

func (ec *ElasticClient) BoolQuery(index, typ string, query map[string]interface{}, body interface{}, id *string) error {
	q := elastic.NewBoolQuery()
	for k, v := range query {
		q = q.Must(elastic.NewTermQuery(k, v))
	}

	searchResult, err := ec.client.Search().Index(index).Type(typ).Query(q).Size(1).Do()
	if err != nil {
		return err
	}
	if searchResult.Hits.TotalHits == 0 {
		return fmt.Errorf("can not find result")
	}
	hit := searchResult.Hits.Hits[0]
	err = json.Unmarshal(*hit.Source, body)
	if err != nil {
		return err
	}

	*id = hit.Id
	return nil
}

func (ec *ElasticClient) BoolQuerys(index, typ string, query map[string]interface{}) (*elastic.SearchResult, error) {
	q := elastic.NewBoolQuery()
	for k, v := range query {
		q = q.Must(elastic.NewTermQuery(k, v))
	}

	return ec.client.Search().Index(index).Type(typ).Query(q).Do()
}

func (ec *ElasticClient) WildcardQuery(index, typ string, key, value string) (*elastic.SearchResult, error) {
	q := elastic.NewWildcardQuery(key, value)
	searchResult, err := ec.client.Search().
		Index(index).
		Type(typ). // search in index "twitter"
		Query(q).  // use wildcard query defined above
		Size(100000000).
		Do() // execute
	if err != nil {
		return nil, err
	}

	return searchResult, nil
}

func (ec *ElasticClient) RestoreByFilename(hc *hdfs.HdfsClient, filename string) error {

	s, err := hc.ReadFile(filename)
	if err != nil {
		return err
	}

	data := bytes.Split(s, []byte("\r\n"))
	for _, d := range data {
		val := bytes.Split(d, []byte("<-|->"))
		if len(val) < 4 {
			continue
		}

		_, err := ec.client.Index().Index(string(val[0])).Type(string(val[1])).Id(string(val[2])).BodyJson(string(val[3])).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func (ec *ElasticClient) Backup(hc *hdfs.HdfsClient, backup_path, node_name string, backup_time int64) error {
	os.MkdirAll(backup_path, os.ModeDir)
	q := elastic.NewWildcardQuery("vpr_utt_node", node_name)
	searchResult, err := ec.client.Search().
		Index(_const.ELASTIC_INDEX).
		Type(_const.ELASTIC_INDEX). // search in index "twitter"
		Query(q).                   // use wildcard query defined above
		Size(100000000).
		Do() // execute
	if err != nil {
		return err
	}

	s := make([][]byte, searchResult.Hits.TotalHits)
	for index, hit := range searchResult.Hits.Hits {
		str := hit.Index + "<-|->" + hit.Type + "<-|->" + hit.Id + "<-|->" + string(*hit.Source)
		s[index] = []byte(str)
	}

	data := bytes.Join(s, []byte("\r\n"))
	return hc.WriteFile(backup_path+"/"+node_name+"_"+strconv.FormatInt(backup_time, 10), data)
}
