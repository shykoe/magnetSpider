package main

import (
	"context"
	"fmt"
	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/shykoe/magnetSpider/dao"
	"github.com/shykoe/magnetSpider/magnet"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"time"
)
const LOG_FILE = "./torrent.log"

func main() {
	var err error
	var addr string
	var port uint16
	var peers int
	var timeout time.Duration
	var verbose bool
	var friends int
	var dsn string
	config := make(map[string]string)
	data, err := ioutil.ReadFile("./config.yml")
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}
	dsn = fmt.Sprintf("mongodb://%s:%s@%s", config["user_name"], config["pass_word"], config["db_addr"])
	fmt.Println(dsn)
	cli, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(dsn))
	esConf := &elasticsearch7.Config{
		Addresses:             []string{
			config["es_addr"],
		},
		Username:              config["es_user"],
		Password:              config["es_pswd"],
	}
	es, err := elasticsearch7.NewClient(*esConf)
	var dao dao.TorrentDaoImpl
	dao.DB = cli
	dao.DataBase = "magnet"
	dao.Es = es
	root := &cobra.Command{
		Use:          "magnetSpider",
		Short:        "magnetSpider - A sniffer that sniffs torrents from BitTorrent network.",
		SilenceUsage: true,
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {

		if err != nil {
			return err
		}
		file, err := os.OpenFile(LOG_FILE, os.O_WRONLY | os.O_CREATE | os.O_APPEND, 0755)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(file)
		if verbose {
			log.SetOutput(os.Stdout)
		}

		p := &magnet.SpiderCore{
			Laddr:      net.JoinHostPort(addr, strconv.Itoa(int(port))),
			Timeout:    timeout,
			MaxFriends: friends,
			MaxPeers:   peers,
			Secret:     string(magnet.RandBytes(20)),
			Dir:        ".",
			Blacklist:  magnet.NewBlackList(5*time.Minute, 50000),
			Dao:        dao,
			Source:     config["source"],
		}
		return p.Run()
	}
	root.Flags().StringVarP(&addr, "addr", "a", "", "listen on given address (default all, ipv4 and ipv6)")
	root.Flags().Uint16VarP(&port, "port", "p", 6881, "listen on given port")
	root.Flags().IntVarP(&friends, "friends", "f", 500, "max fiends to make with per second")
	root.Flags().IntVarP(&peers, "peers", "e", 400, "max peers to connect to download torrents")
	root.Flags().DurationVarP(&timeout, "Timeout", "t", 10*time.Second, "max time allowed for downloading torrents")
	root.Flags().BoolVarP(&verbose, "verbose", "v", false, "Run in verbose mode")

	if err := root.Execute(); err != nil {
		fmt.Println(fmt.Errorf("could not start: %s", err))
	}
}
