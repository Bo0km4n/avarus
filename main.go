package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Bo0km4n/avarus/crawler"
	"github.com/sirupsen/logrus"
)

var (
	baseURL    string
	crawlDepth int
	outputPath string
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s: %s [OPTIONS] ARGS... Options\n`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&baseURL, "root", "https://www.apple.com/", "base url")
	flag.StringVar(&outputPath, "o", "output", "output directory")
	flag.IntVar(&crawlDepth, "depth", 0, "search depth")
	flag.Parse()
}

func main() {
	crawler.SetExecutor(baseURL, crawlDepth, outputPath)
	logrus.Info("Launched scraping...")
	crawler.Run()
}
