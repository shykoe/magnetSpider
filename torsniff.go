package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/marksamman/bencode"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"go.etcd.io/etcd/pkg/fileutil"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	directory = "torrents"
)
var DB *sql.DB
var (
    USERNAME string
    PASSWORD string
    NETWORK  = "tcp"
    SERVER   string
    PORT     int
	DATABASE = "megnet"
	LOCALHOST string
	Count int64
)
type tfile struct {
	name   string
	length int64
}

func (t *tfile) String() string {
	return fmt.Sprintf("name: %s\n, size: %d\n", t.name, t.length)
}

type torrent struct {
	infohashHex string
	name        string
	length      int64
	files       []*tfile
}

func (t *torrent) String() string {
	return fmt.Sprintf(
		"link: %s\nname: %s\nsize: %d\nfile: %d\n",
		fmt.Sprintf("magnet:?xt=urn:btih:%s", t.infohashHex),
		t.name,
		t.length,
		len(t.files),
	)
}
func (t *torrent) InsertDB() error {
	log.Print("Insert ", t.infohashHex, "name ", t.name)
	row, err := DB.Exec(
		"insert INTO t_torrent_info(`hash`,`name`,`discover_time`, `discover_from`) values(?,?,?,?)",
		t.infohashHex, t.name, time.Now(), LOCALHOST)
	if err != nil {
		merr := err.(*mysql.MySQLError)
		if merr.Number == 1062 {
			return errors.New("Dup key!")
		}
		return err
	}
	hashID, _ := row.LastInsertId()
	for _, file := range t.files {
		_, err = DB.Exec("INSERT INTO files_table(`info_id`,`hash`,`file_name`,`size`) values(?,?,?,?)",
			hashID, t.infohashHex, file.name, file.length)
		if err != nil {
			continue
		}
	}
	return nil

}
func parseTorrent(meta []byte, infohashHex string) (*torrent, error) {
	dict, err := bencode.Decode(bytes.NewBuffer(meta))
	if err != nil {
		return nil, err
	}

	t := &torrent{infohashHex: infohashHex}
	if name, ok := dict["name.utf-8"].(string); ok {
		t.name = name
	} else if name, ok := dict["name"].(string); ok {
		t.name = name
	}
	if length, ok := dict["length"].(int64); ok {
		t.length = length
	}

	var totalSize int64
	var extractFiles = func(file map[string]interface{}) {
		var filename string
		var filelength int64
		if inter, ok := file["path.utf-8"].([]interface{}); ok {
			name := make([]string, len(inter))
			for i, v := range inter {
				name[i] = fmt.Sprint(v)
			}
			filename = strings.Join(name, "/")
		} else if inter, ok := file["path"].([]interface{}); ok {
			name := make([]string, len(inter))
			for i, v := range inter {
				name[i] = fmt.Sprint(v)
			}
			filename = strings.Join(name, "/")
		}
		if length, ok := file["length"].(int64); ok {
			filelength = length
			totalSize += filelength
		}
		t.files = append(t.files, &tfile{name: filename, length: filelength})
	}

	if files, ok := dict["files"].([]interface{}); ok {
		for _, file := range files {
			if f, ok := file.(map[string]interface{}); ok {
				extractFiles(f)
			}
		}
	}

	if t.length == 0 {
		t.length = totalSize
	}
	if len(t.files) == 0 {
		t.files = append(t.files, &tfile{name: t.name, length: t.length})
	}

	return t, nil
}

type torsniff struct {
	laddr      string
	maxFriends int
	maxPeers   int
	secret     string
	timeout    time.Duration
	blacklist  *blackList
	dir        string
}

func (t *torsniff) run() error {
	tokens := make(chan struct{}, t.maxPeers)

	dht, err := newDHT(t.laddr, t.maxFriends)
	if err != nil {
		return err
	}

	dht.run()

	log.Println("running, it may take a few minutes...")

	for {
		select {
		case <-dht.announcements.wait():
			for {
				if ac := dht.announcements.get(); ac != nil {
					tokens <- struct{}{}
					go t.work(ac, tokens)
					continue
				}
				break
			}
		case <-dht.die:
			return dht.errDie
		}
	}

}

func (t *torsniff) work(ac *announcement, tokens chan struct{}) {
	defer func() {
		<-tokens
	}()

	if t.isTorrentExist(ac.infohashHex) {
		return
	}

	peerAddr := ac.peer.String()
	if t.blacklist.has(peerAddr) {
		return
	}

	wire := newMetaWire(string(ac.infohash), peerAddr, t.timeout)
	defer wire.free()

	meta, err := wire.fetch()
	if err != nil {
		t.blacklist.add(peerAddr)
		log.Print("wire.fetch() Error ",err)
		return
	}

	// if err := t.saveTorrent(ac.infohashHex, meta); err != nil {
	// 	return
	// }

	torrent, err := parseTorrent(meta, ac.infohashHex)
	if err != nil {
		log.Print("parseTorrent Error")
		return
	}
	err = torrent.InsertDB()
	if err != nil{
		log.Print(err)
	}
	Count ++
	//log.Println(torrent)
}
func (t *torsniff) isTorrentExist(infohashHex string) bool {
    var sum int64
    query := "select count(*) from infohash_table where hashinfo=?"
    if  err := DB.QueryRow(query,infohashHex).Scan(&sum); err != nil{
		log.Fatal(err)
        return true
    }
    if sum == 0{
		log.Print("find New !\n")
        return false
    }
    return true

}

func (t *torsniff) saveTorrent(infohashHex string, data []byte) error {
	name, dir := t.torrentPath(infohashHex)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	d, err := bencode.Decode(bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	f, err := fileutil.TryLockFile(name, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(bencode.Encode(map[string]interface{}{
		"info": d,
	}))
	if err != nil {
		return err
	}

	return nil
}

func (t *torsniff) torrentPath(infohashHex string) (name string, dir string) {
	dir = path.Join(t.dir, infohashHex[:2], infohashHex[len(infohashHex)-2:])
	name = path.Join(dir, infohashHex+".torrent")
	return
}

func main() {
	log.SetFlags(0)
	var err  error
	config := make(map[string] string)
	data, err := ioutil.ReadFile("./config.yml")
	err = yaml.Unmarshal(data,&config)
	if err != nil{
		log.Fatal(err)
	}
	USERNAME = config["username"]
	PASSWORD = config["password"]
	SERVER = config["server"]
	LOCALHOST = config["localname"]
	PORT,_ = strconv.Atoi(config["port"])
	log.Print("USERNAME",USERNAME)
	log.Print("PASSWORD",PASSWORD)
	log.Print("SERVER",SERVER)
	log.Print("LOCALHOST",LOCALHOST)

	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s",USERNAME,PASSWORD,NETWORK,SERVER,PORT,DATABASE)
	log.Printf("dsb %s", dsn)
	DB, err = sql.Open("mysql",dsn)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Print("DB connect ok!")
	DB.SetConnMaxLifetime(100*time.Second)  //最大连接周期，超过时间的连接就close
  	DB.SetMaxOpenConns(1000)//设置最大连接数
	DB.SetMaxIdleConns(0) //设置闲置连接数

	var addr string
	var port uint16
	var peers int
	var timeout time.Duration
	var dir string
	var verbose bool
	var friends int

	home, err := homedir.Dir()
	userHome := path.Join(home, directory)

	root := &cobra.Command{
		Use:          "torsniff",
		Short:        "torsniff - A sniffer that sniffs torrents from BitTorrent network.",
		SilenceUsage: true,
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if dir == userHome && err != nil {
			return err
		}

		absDir, err := filepath.Abs(dir)
		if err != nil {
			return err
		}

		log.SetOutput(ioutil.Discard)
		if verbose {
			log.SetOutput(os.Stdout)
		}

		p := &torsniff{
			laddr:      net.JoinHostPort(addr, strconv.Itoa(int(port))),
			timeout:    timeout,
			maxFriends: friends,
			maxPeers:   peers,
			secret:     string(randBytes(20)),
			dir:        absDir,
			blacklist:  newBlackList(5*time.Minute, 50000),
		}
		return p.run()
	}

	root.Flags().StringVarP(&addr, "addr", "a", "", "listen on given address (default all, ipv4 and ipv6)")
	root.Flags().Uint16VarP(&port, "port", "p", 6881, "listen on given port")
	root.Flags().IntVarP(&friends, "friends", "f", 500, "max fiends to make with per second")
	root.Flags().IntVarP(&peers, "peers", "e", 400, "max peers to connect to download torrents")
	root.Flags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "max time allowed for downloading torrents")
	root.Flags().StringVarP(&dir, "dir", "d", userHome, "the directory to store the torrents")
	root.Flags().BoolVarP(&verbose, "verbose", "v", false, "run in verbose mode")

	if err := root.Execute(); err != nil {
		fmt.Println(fmt.Errorf("could not start: %s", err))
	}
}
