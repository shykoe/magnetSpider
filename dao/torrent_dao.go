package dao

import (
	"context"
	"encoding/json"
	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/shykoe/magnetSpider/model"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
	"time"
)

type TorrentDao interface {
	InsertTorrent(torrent *model.Torrent) error
	TorrentNotExists(infoHash string) bool
	InsertTorrent2Es(torrent *model.Torrent) error
}
type TorrentDaoImpl struct {
	DB       *mongo.Client
	DataBase string
	Es       *elasticsearch7.Client
}

func (t TorrentDaoImpl) InsertTorrent(torrent *model.Torrent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	collection := t.DB.Database(t.DataBase).Collection("torrent")
	doc, err := torrent.ToDoc()
	if err != nil {
		return err
	}
	_, err = collection.InsertOne(ctx, doc)
	defer cancel()
	if err != nil {
		return err
	}
	return nil
}

func (t TorrentDaoImpl) TorrentNotExists(infoHash string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	collection := t.DB.Database(t.DataBase).Collection("torrent")
	filter := bson.D{
		{"infohash", infoHash},
	}
	defer cancel()
	var result bson.M
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false
		}
		log.Errorf("IsTorrentExists err %s", err.Error())
		return false
	}
	return true
}

func (t TorrentDaoImpl) InsertTorrent2Es(torrent *model.Torrent) error {
	if t.Es == nil{
		log.Error("es nil ")
		return nil
	}
	bytes, err := json.Marshal(torrent)
	if err != nil{
		log.Error(err.Error())
		return err
	}
	req := esapi.IndexRequest{
		Index:      "torrent_1",
		DocumentID: torrent.InfoHash,
		Body:       strings.NewReader(string(bytes)),
		Refresh:    "false",
	}
	res, err := req.Do(context.Background(), t.Es)
	if err != nil {
		log.Errorf("Error getting response: %s", err)
	}
	defer res.Body.Close()
	if res.IsError(){
		log.Errorf("[%s] Error indexing document ID=%d msg=%s", res.Status(), torrent.InfoHash, res.String())
	}
	return nil
}