package main

import "github.com/sirupsen/logrus"

// Context クロール用状態Context
type Context struct {
	BaseURL    string
	Depth      int
	OutputPath string
}

// Run 始動関数
func (ctx *Context) Run() error {
	page := NewPage(ctx.BaseURL)
	page.Init()
	page.FetchFiles()
	page.RewriteDoc()
	page.WriteHTML()
	logrus.Info("Done!")
	return nil
}
