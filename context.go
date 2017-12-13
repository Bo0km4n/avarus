package main

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Context クロール用状態Context
type Context struct {
	BaseURL    string
	Depth      int
	OutputPath string
	Refferer   map[string]string
	Pages      []Page
}

// NewContext context コンストラクト
func NewContext(url string, depth int, outputPath string) Context {
	ctx := Context{
		BaseURL:    url,
		Depth:      depth,
		OutputPath: outputPath,
		Refferer:   map[string]string{},
		Pages:      []Page{},
	}
	return ctx
}

// Run 始動関数
func (ctx *Context) Run() error {
	page := NewPage(ctx.BaseURL, 1)
	page.Init()
	page.QueuingPages()
	page.FetchFiles()
	page.RewriteDoc()
	page.WriteHTML()

	// async
	var wg sync.WaitGroup
	for _, v := range ctx.Pages {
		wg.Add(1)
		//pp.Println(v)
		go func(v Page) {
			v.Init()
			v.QueuingPages()
			v.FetchFiles()
			v.RewriteDoc()
			v.WriteHTML()
			wg.Done()
		}(v)
	}
	wg.Wait()
	logrus.Info("Done!")
	return nil
}
