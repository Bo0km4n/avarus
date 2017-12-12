package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	baseURL    string
	crawlDepth int
	outputPath string
	ctx        Context
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage of %s: %s [OPTIONS] ARGS... Options`, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.StringVar(&baseURL, "b", "https://www.apple.com/", "base url")
	flag.StringVar(&outputPath, "o", "output", "output directory")
	flag.IntVar(&crawlDepth, "d", 1, "search depth")
	flag.Parse()

}

func main() {
	ctx.BaseURL = baseURL
	ctx.Depth = crawlDepth
	ctx.OutputPath = outputPath
	logrus.Info("Launched scraping...")
	ctx.Run()
	//cssParse()
}
