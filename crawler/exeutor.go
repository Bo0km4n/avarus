package crawler

import (
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Executor
type Executor struct {
	RootURL    string
	Depth      int
	OutputPath string
	Refferer   map[string]string
	Pages      []Page
}

var E *Executor

func SetExecutor(url string, depth int, outputPath string) {
	E = &Executor{
		RootURL:    url,
		Depth:      depth,
		OutputPath: outputPath,
		Refferer:   map[string]string{},
		Pages:      []Page{},
	}
}

// Run is entry point
func Run() error {
	start := time.Now()
	page := NewPage(E.RootURL, 1)
	page.Exec()

	cpus := runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
	c := make(chan bool, cpus)
	// async
	for i := 0; i < E.Depth; i++ {
		var wg sync.WaitGroup
		for _, p := range E.Pages {
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
	logrus.Infof("Result: %d pages, %f seconds\n", len(E.Refferer), (end.Sub(start)).Seconds())
	return nil
}
