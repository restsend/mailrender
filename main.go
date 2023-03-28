package main

import (
	"flag"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

var httpServerAddr string
var localServerAddr string
var storeDir string
var author string

// Mb
var uploadLimit int

func main() {
	log.Default().SetFlags(log.Lshortfile | log.LstdFlags)

	flag.StringVar(&httpServerAddr, "http", "0.0.0.0:8000", "HTTP listen addr")
	flag.StringVar(&storeDir, "store", "/tmp/mailrender/", "Upload store directory")
	flag.IntVar(&uploadLimit, "m", 50, "Size limit (MB)")
	flag.StringVar(&author, "author", "restsend.com", "Exif/author meta info")

	flag.Parse()

	err := prepareStoreDir(storeDir)
	if err != nil {
		panic(err)
	}
	if strings.HasPrefix(httpServerAddr, ":") {
		localServerAddr = "http://127.0.0.1" + httpServerAddr
	} else {
		localServerAddr = strings.Replace(httpServerAddr, "0.0.0.0", "http://127.0.0.1", 1)
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	log.Println("Starting HTTP Server at http://"+httpServerAddr, "localaddr at", localServerAddr)

	RegisterHandlers(r)
	r.Run(httpServerAddr)
}
