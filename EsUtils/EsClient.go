package EsUtils

import (
	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"log"

)
var esConf *elasticsearch7.Config
var esClient *elasticsearch7.Client
//获取es client
func GetESClient()(*elasticsearch7.Client,error)  {
	if esClient!=nil{
		resp, err := esClient.Ping()
		log.Print(resp)
		if err == nil && resp.StatusCode == 200{
			return esClient,nil
		}
	}
	client, err := elasticsearch7.NewClient(elasticsearch7.Config{
		Addresses:             []string{
			"[2400:dd01:1032:f032:ec4:7aff:feb3:2c10]:9200",
		},
		Username:              "elastic",
		Password:              "Shykoetozx!2",
	})
	if err != nil{
		return nil, err
	}
	esClient = client
	return esClient,nil
}