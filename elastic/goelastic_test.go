package elastic

import (
	"fmt"
	"git.oschina.net/kuaishangtong/asvserver/models"
	"gopkg.in/olivere/elastic.v2"
	"testing"
	"time"
)

type Student struct {
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Gender string `json:"gender"`
}

func TestElasticClient_IndexExists(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16"}, []string{"9200"})
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
	client, err := NewElasticClient([]string{"192.168.1.16"}, []string{"9200"})
	if err != nil {
		t.Fatal(err)
	}

	var __ASV_VPR_INFO_INDEX string = `{
  "asv_vpr_info": {
    "mappings": {
      "asv_vpr_info": {
        "_ttl": {
          "enabled": false
        },
        "_timestamp": {
          "enabled": true
        },
        "_all": {
          "enabled": false
        },
        "properties": {
          "vpr_task_id": {
            //注册该条语音的任务ID
            "index": "not_analyzed",
            "store": "yes",
            "type": "string"
          },
          "vpr_spk_id": {
            //语音唯一表示
            "index": "not_analyzed",
            "store": "yes",
            "type": "string"
          },
          "vpr_wav_file": {
            //语音文件名
            "index": "not_analyzed",
            "store": "yes",
            "type": "string"
          },
          "vpr_utt_node": {
            //声纹库id
            "index": "not_analyzed",
            "store": "yes",
            "type": "string"
          },
          "vpr_add_time": {
            //入库时间
            "index": "not_analyzed",
            "store": "yes",
            "type": "date"
          },
          "vpr_utt_duration": {
            //语音时长
            "index": "not_analyzed",
            "store": "yes",
            "type": "integer"
          },
          "vpr_utt_valid_dura": {
            //有效时长
            "index": "not_analyzed",
            "store": "yes",
            "type": "integer"
          },
          "vpr_utt_chan": {
            //声道
            "index": "not_analyzed",
            "store": "yes",
            "type": "integer"
          },
          "vpr_utt_dir": {
            //语音路径
            "index": "not_analyzed",
            "store": "yes",
            "type": "string"
          },
          "vpr_utt_gender": {
            //说话人性别
            "index": "not_analyzed",
            "store": "yes",
            "type": "string"
          },
          "vpr_norm_params": {
            // 得分归一化参数
            "index": "not_analyzed",
            "store": "yes",
            "type": "integer"
          },
          "vpr_utt_scene": {
            // 语音场景
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
}
`

	res, err := client.CreateIndexBodyString("asv_vpr_info", __ASV_VPR_INFO_INDEX)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res.Acknowledged)
}

func TestElasticClient_DeleteIndex(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16"}, []string{"9200"})
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.DeleteIndex("asv_vpr_info")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res.Acknowledged)
}

func TestElasticClient_IndexBodyJson(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16"}, []string{"9200"})
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

	vpr_info.SetVprAddTime(time.Now().Format("2006-01-02 15:04:05"))
	vpr_info.SetVprWavFile("tbfaa_A.wav")

	res, err := client.InsertDocBodyJsonWithID("asv_vpr_info", "asv_vpr_info", "3234567890ABCDEF", vpr_info)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res)
}

func TestElasticClient_BoolQuery(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16"}, []string{"9200"})
	if err != nil {
		t.Fatal(err)
	}

	index, typ := "asv_voiceprint_info", "asv_voiceprint_info"

	query := make(map[string]interface{})
	query["vpr_utt_recordid"] = "*"
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
	client, err := NewElasticClient([]string{"192.168.1.16"}, []string{"9200"})
	if err != nil {
		t.Fatal(err)
	}

	// Define wildcard query
	q := elastic.NewWildcardQuery("vpr_utt_recordid", "*")
	searchResult, err := client.client.Search().
		Index("asv_voiceprint_info").
		Type("asv_voiceprint_info"). // search in index "twitter"
		Query(q).                    // use wildcard query defined above
		Do()                         // execute
	if err != nil {
		// Handle error
		panic(err)
	}

	t.Log(searchResult.Hits.TotalHits)
}

func TestElasticClient_UpdateDocBodyWithID(t *testing.T) {
	client, err := NewElasticClient([]string{"192.168.1.16"}, []string{"9200"})
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
