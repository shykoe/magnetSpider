package dao

import (
	"context"
	"github.com/shykoe/magnetSpider/model"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

type TorrentDao interface {
	InsertTorrent(torrent *model.Torrent) error
	TorrentNotExists(infoHash string) bool
}
type TorrentDaoImpl struct {
	DB       *mongo.Client
	DataBase string
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
			return true
		}
		log.Errorf("IsTorrentExists err %s", err.Error())
		return false
	}
	return false
}
