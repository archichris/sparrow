package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	// "sync"

	"sparrow/auth"
	"sparrow/cfg"
	_ "sparrow/etcdv3"
	"sparrow/proxy"
	_ "sparrow/session/providers/memory"
)

var (
	rootPath = flag.String("root", "", "root directory")
)

func main() {
	auth.Test()
	cfg.Test()
	proxy.Test()
	// etcdv3.Test()
	host := flag.String("h", "0.0.0.0", "host name or ip address")
	port := flag.Int("p", 8080, "port")

	flag.CommandLine.Parse(os.Args[1:])

	if len(*rootPath) == 0 {
		path, err := os.Executable()
		if err != nil {
			*rootPath = "./assets"
		} else {
			*rootPath = filepath.Join(filepath.Dir(path), "assets")
		}
	}

	http.Handle("/", &auth.AuthHandler{RootPath: *rootPath}) // view static directory

	log.Printf("listening on %s:%d\n", *host, *port)
	err := http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
