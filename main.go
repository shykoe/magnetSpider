package main

import (
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	var err  error
	var addr string
	var port uint16
	var peers int
	var timeout time.Duration
	var dir string
	var verbose bool
	var friends int
	config := make(map[string] string)
	data, err := ioutil.ReadFile("./config.yml")
	err = yaml.Unmarshal(data,&config)
	if err != nil{
		log.Fatal(err)
	}
	root := &cobra.Command{
		Use:          "magnetSpider",
		Short:        "magnetSpider - A sniffer that sniffs torrents from BitTorrent network.",
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

		p := & torsniff{
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
}