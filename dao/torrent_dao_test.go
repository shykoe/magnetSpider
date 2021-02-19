package dao

import (
	"context"
	"fmt"
	"github.com/shykoe/magnetSpider/model"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"testing"
	"time"
)

func TestTorrentDaoImpl_InsertTorrent(t1 *testing.T) {
	var files []*model.TorrentFile
	for i := 1; i < 10; i++ {
		file := model.TorrentFile{
			Name: "12",
			Size: int64(i),
		}
		files = append(files, &file)
	}
	torr := model.Torrent{
		InfoHash: "aasasdas",
		Name:     "asdasdas",
		Length:   100,
		Files:    files,
		DiscoverFrom: "local",
		DiscoverTime: time.Now(),
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(""))
	if err != nil{
		log.Fatal(err)
	}
	var dao TorrentDaoImpl
	dao.DataBase = "magnet"
	dao.DB = cli
	dao.InsertTorrent(&torr)
}

func TestTorrentDaoImpl_IsTorrentNotExists(t1 *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(""))
	if err != nil{
		log.Fatal(err)
	}
	var dao TorrentDaoImpl
	dao.DB = cli
	dao.DataBase = "magnet"
	v := dao.TorrentNotExists("aasasdas")
	fmt.Println(v)
}