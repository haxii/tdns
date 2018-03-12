package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/haxii/tdns"
)

var (
	rpc    = flag.String("rpc", "127.0.0.1:8090", "rpc server addr")
	host   = flag.String("host", "", "host to resolve")
	peerIP = flag.String("peerIP", "61.135.169.125", "peer ip")
	help   = flag.Bool("h", false, "help usage")
)

func main() {
	flag.Parse()
	if *help {
		flag.Usage()
		return
	}

	if *rpc == "" || *host == "" {
		log.Fatalln("rpc or host is empty")
	}

	c := tdns.ConnectClient(*rpc)
	defer c.Close()

	res, err := c.LookupIPAddr(*peerIP, *host)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%s-->%+v\n", *host, res)
}
