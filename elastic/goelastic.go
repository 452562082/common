package elastic

import (
	"encoding/json"
	"fmt"
	"gopkg.in/olivere/elastic.v2"
	"bytes"
	"io/ioutil"
	"git.oschina.net/kuaishangtong/asvWebApi/const"
	"strconv"
	"os"
)

var ASV_VPR_INFO_INDEX string = `{
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

	client, err := elastic.NewClient(elastic.SetURL(urls...))
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
		Size(10000000).
		Do() // execute
	if err != nil {
		return nil, err
	}

	return searchResult, nil
}

/*func (ec *ElasticClient) BackupByNodename(index, _type, node_name, filename string) error {
	//client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	//if err != nil {
	//	t.Fatal(err)
	//}

	q := elastic.NewWildcardQuery("vpr_utt_node", node_name)
	searchResult, err := ec.client.Search().
		Index(index).
		Type(_type). // search in index "twitter"
		Query(q).                    // use wildcard query defined above
		Size(10000000).
		Do()                         // execute
	if err != nil {
		// Handle error
		return err
	}

	//t.Logf("totalHits: %d",searchResult.Hits.TotalHits)

	//filename := "D:\\backup.txt"

	s := make([][]byte, searchResult.Hits.TotalHits)
	for index, hit := range searchResult.Hits.Hits {
		if err != nil {
			// Deserialization failed
			//t.Fatal(err)
			return err
		}
		str := hit.Index + "<-|->" + hit.Type + "<-|->" + hit.Id + "<-|->" + string(*hit.Source)
		s[index] = []byte(str)
	}
	data := bytes.Join(s, []byte("\r\n"))
	ioutil.WriteFile(filename, data, 0666)
	return nil
}*/

func (ec *ElasticClient) RestoreByFilename(filename string)  error{
	//filename := "D:\\backup.txt"
	s, err := ioutil.ReadFile(filename)
	if err != nil {
		//t.Fatal(err)
		return err
	}

	data := bytes.Split(s, []byte("\r\n"))
	for _, d := range data {
		val := bytes.Split(d,[]byte("<-|->"))
		if len(val) < 4 {
			continue
		}

		//fmt.Println(string(val[0]))
		//fmt.Println(string(val[1]))
		//fmt.Println(string(val[2]))
		//fmt.Println(string(val[3]))
		_, err := ec.client.Index().Index(string(val[0])).Type(string(val[1])).Id(string(val[2])).BodyJson(string(val[3])).Do()
		if err != nil {
			//t.Fatal(err)
			return err
		}
	}
	return nil
}

func (ec *ElasticClient) Backup(backup_path, node_name string, backup_time int64) error {
	//backup_path := "/tmp/backup/"
	os.MkdirAll(backup_path, os.ModeDir)
	//__elastic_client.BackupByNodename(_const.ELASTIC_INDEX, _const.ELASTIC_INDEX, backup.Lib.LibNodeId, backup_path + backup.Lib.LibNodeId + "_" + strconv.FormatInt(backup.BackupTime,10))
	q := elastic.NewWildcardQuery("vpr_utt_node", node_name)
	searchResult, err := ec.client.Search().
		Index(_const.ELASTIC_INDEX).
		Type(_const.ELASTIC_INDEX). // search in index "twitter"
		Query(q).                    // use wildcard query defined above
		Size(10000000).
		Do()                         // execute
	if err != nil {
		// Handle error
		return err
	}


	s := make([][]byte, searchResult.Hits.TotalHits)
	for index, hit := range searchResult.Hits.Hits {
		if err != nil {
			// Deserialization failed
			//t.Fatal(err)
			return err
		}
		str := hit.Index + "<-|->" + hit.Type + "<-|->" + hit.Id + "<-|->" + string(*hit.Source)
		s[index] = []byte(str)
	}
	data := bytes.Join(s, []byte("\r\n"))
	ioutil.WriteFile(backup_path + node_name + "_" + strconv.FormatInt(backup_time,10), data, 0666)

	return nil
}