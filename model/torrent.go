package model

import (
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

type TorrentFile struct {
	Name string `json:"Name"`
	Size int64  `json:"Size"`
}
type Torrent struct {
	InfoHash string         `json:"Hash"`
	Name     string         `json:"Name"`
	Length   int64          `json:"Length"`
	DiscoverFrom string `json:"DiscoverFrom"`
	DiscoverTime time.Time `json:"DiscoverTime"`
	Files    []*TorrentFile `json:"Files"`
}

func (t *Torrent) ToJson() string {
	jsonBytes, _ := json.Marshal(t)
	return string(jsonBytes)
}
func (t *Torrent) ToDoc() (doc *bson.D, err error) {
	data, err := bson.Marshal(t)
	if err != nil {
		return
	}
	err = bson.Unmarshal(data, &doc)
	return
}
