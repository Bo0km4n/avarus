package crawler

import (
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Context クロール用状態Context
type Context struct {
	RootURL    string
	Depth      int
	OutputPath string
	Refferer   map[string]string
	Pages      []Page
}

// Ctx is Cralwer context
var Ctx *Context

// NewContext context コンストラクト
func NewContext(url string, depth int, outputPath string) *Context {
	return &Context{
		RootURL:    url,
		Depth:      depth,
		OutputPath: outputPath,
		Refferer:   map[string]string{},
		Pages:      []Page{},
	}
}

// Run 始動関数
func (ctx *Context) Run() error {
	start := time.Now()
	page := NewPage(ctx.RootURL, 1)
	page.Exec()

	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	c := make(chan bool, cpus)
	// async
	for i := 0; i < ctx.Depth; i++ {
		var wg sync.WaitGroup
		for _, p := range ctx.Pages {
			c <- true
			wg.Add(1)
			go func(p Page) {
				defer func() { <-c }()
				p.Exec()
				wg.Done()
			}(p)
		}
		wg.Wait()
	}
	end := time.Now()
	logrus.Infof("Result: %d pages, %f seconds\n", len(Ctx.Refferer), (end.Sub(start)).Seconds())
	return nil
}
