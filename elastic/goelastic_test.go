package elastic

import (
	"fmt"
	"kuaishangtong/asvserver/models"
	"gopkg.in/olivere/elastic.v2"
	"testing"
	"time"
	//"encoding/json"
	"io/ioutil"
	"bytes"
	"strings"
)

type Student struct {
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Gender string `json:"gender"`
}

func TestElasticClient_IndexExists(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	exist, err := client.IndexExists("asv_vpr_info")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(exist)
}

func TestElasticClient_CreateIndexBodyString(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	var __ASV_VPR_INFO_INDEX string =`{
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
          "min_gram": 3,
          "max_gram": 3,
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
      "asv_voiceprint_info": {
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

	res, err := client.CreateIndexBodyString("asv_voiceprint_info", __ASV_VPR_INFO_INDEX)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res.Acknowledged)
}

func TestElasticClient_DeleteIndex(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.client.DeleteIndex("asv_voiceprint_info&2017-11-10").Do()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res.Acknowledged)
}

func TestElasticClient_IndexBodyJson(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	vpr_info := models.NewAsvVprInfo()
	vpr_info.SetVprSpkId("702c0afb3d424a70")
	vpr_info.SetVprUttDir("hdfs:/kaldi/kaldi/kaldi/src/kvpbin/data/")
	vpr_info.SetVprUttDuration(fmt.Sprintf("%f", 300.00))
	vpr_info.SetVprUttValidDura(fmt.Sprintf("%f", 210.00))
	vpr_info.SetVprUttChan(fmt.Sprintf("%d", 1))
	vpr_info.SetVprUttNode("node")
	vpr_info.SetVprUttGender("M")
	vpr_info.SetVprUttScene("lcc")
	vpr_info.SetVprHasTone("no")
	vpr_info.SetVprRecordId("recordId-testnode-00001")
	vpr_info.SetVprAddTime(time.Now().Format("2006-01-02 15:04:05"))
	vpr_info.SetVprWavFile("tbfaa_A.wav")

	res, err := client.InsertDocBodyJsonWithID("asv_voiceprint_info", "asv_voiceprint_info", "3234567890ABCDEF", vpr_info)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res)
}

func TestElasticClient_BoolQuery(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.17:9200"})
	if err != nil {
		t.Fatal(err)
	}

	index, typ := "asv_voiceprint_info", "asv_voiceprint_info"

	query := make(map[string]interface{})
	query["vpr_utt_recordid"] = "recordId-testnode-00001"
	//query["vpr_utt_node"] = "testnode"
	vpr := models.AcquireAsvVprInfo()
	var id string
	err = client.BoolQuery(index, typ, query, vpr, &id)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("recordid: %s", vpr.VPR_RecordId)
}

func TestElasticClient_WildcardQuery(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	// Define wildcard query
	q := elastic.NewWildcardQuery("vpr_utt_recordid", "recordId-*-00001")
	searchResult, err := client.client.Search().
		Index("asv_voiceprint_info").
		Type("asv_voiceprint_info"). // search in index "twitter"
		Query(q).                    // use wildcard query defined above
		Do()                        // execute
	if err != nil {
		// Handle error
		panic(err)
	}

	t.Logf("totalHits: %d",searchResult.Hits.TotalHits)
}

func TestElasticClient_backupNode(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	q := elastic.NewWildcardQuery("vpr_utt_recordid", "*")
	searchResult, err := client.client.Search().
		Index("asv_voiceprint_info&test").
		Type("asv_voiceprint_info&test"). // search in index "twitter"
		Query(q).                    // use wildcard query defined above
		Size(10000).
		Do()                         // execute
	if err != nil {
		// Handle error
		panic(err)
	}

	t.Logf("totalHits: %d",searchResult.Hits.TotalHits)

	filename := "D:\\backup.txt"

	s := make([][]byte, searchResult.Hits.TotalHits)
	for index, hit := range searchResult.Hits.Hits {
		if err != nil {
			// Deserialization failed
			t.Fatal(err)
		}
		str := hit.Index + "<-|->" + hit.Type + "<-|->" + hit.Id + "<-|->" + string(*hit.Source)
		s[index] = []byte(str)
	}
	data := bytes.Join(s, []byte("\r\n"))
	ioutil.WriteFile(filename, data, 0666)
}

func TestElasticClient_backupNode2(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	node_name := "recordId?*"
	q := elastic.NewWildcardQuery("vpr_utt_recordid", node_name)
	searchResult, err := client.client.Search().
		Index("asv_voiceprint_info").
		Type("asv_voiceprint_info"). // search in index "twitter"
		Query(q).                    // use wildcard query defined above
		Size(10000).
		Do()                         // execute
	if err != nil {
		// Handle error
		panic(err)
	}

	t.Logf("totalHits: %d",searchResult.Hits.TotalHits)

	//timeStr := time.Now().Format("2006-01-02")
	timeStr := "test"
	//s := make([][]byte, searchResult.Hits.TotalHits)
	for _, hit := range searchResult.Hits.Hits {
		if err != nil {
			// Deserialization failed
			t.Fatal(err)
		}

		_, err := client.client.Index().Index(hit.Type + "&" + timeStr).Type(hit.Type + "&" + timeStr).Id(hit.Id).BodyJson(hit.Source).Do()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestElasticClient_RestoreNode(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	filename := "D:\\backup.txt"
	s, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	//res, err := client.InsertDocBodyJsonWithID("asv_vpr_info", "asv_vpr_info", "3234567890ABCDEF", vpr_info)
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
		_, err := client.client.Index().Index(string(val[0])).Type(string(val[1])).Id(string(val[2])).BodyJson(string(val[3])).Do()
		if err != nil {
			t.Fatal(err)
		}
	}
	//t.Log(res)
}

func TestElasticClient_RestoreNode2(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	node_name := "testnode"
	q := elastic.NewWildcardQuery("vpr_utt_node", node_name)
	timeStr := time.Now().Format("2006-01-02")
	searchResult, err := client.client.Search().
		Index("asv_voiceprint_info&" + timeStr).
		Type("asv_voiceprint_info&" + timeStr). // search in index "twitter"
		Query(q).                    // use wildcard query defined above
		Size(10000).
		Do()                         // execute
	if err != nil {
		// Handle error
		panic(err)
	}

	//res, err := client.InsertDocBodyJsonWithID("asv_vpr_info", "asv_vpr_info", "3234567890ABCDEF", vpr_info)
	for _, hit := range searchResult.Hits.Hits {
		if err != nil {
			// Deserialization failed
			t.Fatal(err)
		}
		//s[index] = *hit.Source
		_, err := client.client.Index().Index(strings.Split(hit.Index,"&")[0]).Type(strings.Split(hit.Type,"&")[0]).Id(hit.Id).BodyJson(hit.Source).Do()
		if err != nil {
			t.Fatal(err)
		}
	}
	//t.Log(res)
}

func TestElasticClient_UpdateDocBodyWithID(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16:9200"})
	if err != nil {
		t.Fatal(err)
	}

	data := make(map[string]interface{})
	data["vpr_utt_node"] = "testnode1"
	data["_id"] = "1111"
	err = client.UpdateDocBodyWithID("asv_voiceprint_info", "asv_voiceprint_info", "testnode#recordId00019#4659e731868e24a0", data)
	if err != nil {
		t.Fatal(err)
	}
}