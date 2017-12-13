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
	page.Exec()
	// async
	var wg sync.WaitGroup
	for _, v := range ctx.Pages {
		wg.Add(1)
		go func(v Page) {
			v.Exec()
			wg.Done()
		}(v)
	}
	wg.Wait()
	logrus.Info("Done!")
	return nil
}
