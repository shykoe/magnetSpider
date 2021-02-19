package model

import (
	"fmt"
	"testing"
)

func TestTorrentFile_ToJson(t1 *testing.T) {
	var files []*TorrentFile
	for i := 1; i < 10; i++ {
		file := TorrentFile{
			Name: "12",
			Size: int64(i),
		}
		files = append(files, &file)
	}
	torr := Torrent{
		InfoHash: "aasasdas",
		Name:     "asdasdas",
		Length:   100,
		Files:    files,
	}
	println(torr.ToJson())
}

func TestTorrent_ToDoc(t1 *testing.T) {
	var files []*TorrentFile
	for i := 1; i < 10; i++ {
		file := TorrentFile{
			Name: "12",
			Size: int64(i),
		}
		files = append(files, &file)
	}
	torr := Torrent{
		InfoHash: "aasasdas",
		Name:     "asdasdas",
		Length:   100,
		Files:    files,
	}
	d, err := torr.ToDoc()
	if err != nil{
		fmt.Println(err)
	}
	d.Map()
}