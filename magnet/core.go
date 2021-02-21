package magnet

import (
	"bytes"
	"fmt"
	"github.com/marksamman/bencode"
	"github.com/mitchellh/go-homedir"
	"github.com/shykoe/magnetSpider/dao"
	"github.com/shykoe/magnetSpider/model"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.etcd.io/etcd/pkg/fileutil"
	"gopkg.in/yaml.v2"
	"io/ioutil"
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

type Torrent struct {
	infohashHex string
	name        string
	length      int64
	files       []*tfile
}

func (t *Torrent) String() string {
	return fmt.Sprintf(
		"link: %s\nname: %s\nsize: %d\nfile: %d\n",
		fmt.Sprintf("magnet:?xt=urn:btih:%s", t.infohashHex),
		t.name,
		t.length,
		len(t.files),
	)
}
func (t *Torrent) InsertDB() error {
	return nil
}
func parseTorrent(meta []byte, infohashHex string) (*model.Torrent, error) {
	dict, err := bencode.Decode(bytes.NewBuffer(meta))
	if err != nil {
		return nil, err
	}

	t := &model.Torrent{
		InfoHash: infohashHex,
		DiscoverTime: time.Now(),
	}
	if name, ok := dict["name.utf-8"].(string); ok {
		t.Name = name
	} else if name, ok := dict["name"].(string); ok {
		t.Name = name
	}
	if length, ok := dict["length"].(int64); ok {
		t.Length = length
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
		t.Files = append(t.Files, &model.TorrentFile{Name: filename, Size: filelength})
	}

	if files, ok := dict["files"].([]interface{}); ok {
		for _, file := range files {
			if f, ok := file.(map[string]interface{}); ok {
				extractFiles(f)
			}
		}
	}

	if t.Length == 0 {
		t.Length = totalSize
	}
	if len(t.Files) == 0 {
		t.Files = append(t.Files, &model.TorrentFile{Name: t.Name, Size: t.Length})
	}

	return t, nil
}

type SpiderCore struct {
	Laddr      string
	MaxFriends int
	MaxPeers   int
	Secret     string
	Timeout    time.Duration
	Blacklist  *blackList
	Dir        string
	Dao        dao.TorrentDao
}

func (t *SpiderCore) Run() error {
	tokens := make(chan struct{}, t.MaxPeers)

	dht, err := newDHT(t.Laddr, t.MaxFriends)
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

func (t *SpiderCore) work(ac *announcement, tokens chan struct{}) {
	defer func() {
		<-tokens
	}()
	log.Infof("income %s", ac.infohashHex)
	exists := t.isTorrentExist(ac.infohashHex)
	if exists {
		log.Infof("infohash %s exists pass", ac.infohashHex)
		return
	}

	peerAddr := ac.peer.String()
	if t.Blacklist.has(peerAddr) {
		return
	}

	wire := newMetaWire(string(ac.infohash), peerAddr, t.Timeout)
	defer wire.free()

	meta, err := wire.fetch()
	if err != nil {
		t.Blacklist.add(peerAddr)
		log.Print("wire.fetch() Error ",err)
		return
	}



	torrent, err := parseTorrent(meta, ac.infohashHex)
	if err != nil {
		log.Print("parseTorrent Error")
		return
	}
	if torrent != nil {
		err := t.Dao.InsertTorrent(torrent)
		if err != nil {
			log.Errorf("InsertTorrent err: %s", err.Error())
		}
	}
	if err != nil{
		log.Print(err)
	}
	Count ++
}
func (t *SpiderCore) isTorrentExist(infoHash string) bool {

	return t.Dao.TorrentNotExists(infoHash)
}

func (t *SpiderCore) saveTorrent(infohashHex string, data []byte) error {
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

func (t *SpiderCore) torrentPath(infohashHex string) (name string, dir string) {
	dir = path.Join(t.Dir, infohashHex[:2], infohashHex[len(infohashHex)-2:])
	name = path.Join(dir, infohashHex+".Torrent")
	return
}

func main() {

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
		Use:          "SpiderCore",
		Short:        "SpiderCore - A sniffer that sniffs torrents from BitTorrent network.",
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

		p := &SpiderCore{
			Laddr:      net.JoinHostPort(addr, strconv.Itoa(int(port))),
			Timeout:    timeout,
			MaxFriends: friends,
			MaxPeers:   peers,
			Secret:     string(RandBytes(20)),
			Dir:        absDir,
			Blacklist:  NewBlackList(5*time.Minute, 50000),
		}
		return p.Run()
	}

	root.Flags().StringVarP(&addr, "addr", "a", "", "listen on given address (default all, ipv4 and ipv6)")
	root.Flags().Uint16VarP(&port, "port", "p", 6881, "listen on given port")
	root.Flags().IntVarP(&friends, "friends", "f", 500, "max fiends to make with per second")
	root.Flags().IntVarP(&peers, "peers", "e", 400, "max peers to connect to download torrents")
	root.Flags().DurationVarP(&timeout, "Timeout", "t", 10*time.Second, "max time allowed for downloading torrents")
	root.Flags().StringVarP(&dir, "Dir", "d", userHome, "the directory to store the torrents")
	root.Flags().BoolVarP(&verbose, "verbose", "v", false, "Run in verbose mode")

	if err := root.Execute(); err != nil {
		fmt.Println(fmt.Errorf("could not start: %s", err))
	}
}
