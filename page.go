package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/PuerkitoBio/goquery"
)

const (
	httpToken   = "http://"
	httpsToken  = "https://"
	fileTypeCSS = "css"
	fileTypeIMG = "img"
	fileTypeJS  = "js"
)

// Page htmlページ構造体
type Page struct {
	Level        int
	Doc          *goquery.Document
	Domain       string
	DomainScheme string
	Path         string
	URL          string
	UUID         string
	HTML         string
	ImgMap       map[string]string
	CSSMap       map[string]string
	LinkMap      map[string]string
	JSMap        map[string]string
}

// NewPage Page生成
func NewPage(url string) Page {
	page := Page{
		Level:   0,
		URL:     url,
		HTML:    "",
		ImgMap:  map[string]string{},
		LinkMap: map[string]string{},
		JSMap:   map[string]string{},
		CSSMap:  map[string]string{},
	}
	return page
}

// Init 構造体初期化
func (p *Page) Init() {
	doc, err := goquery.NewDocument(p.URL)
	p.Doc = doc
	if err != nil {
		logrus.Fatalf("Failed get document %s", p.URL)
		return
	}
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		if !strings.Contains(link, ".img") && (strings.Contains(link, ".html") || strings.Contains(link, "http")) {
			p.LinkMap[link] = ""
		}
	})
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		js, _ := s.Attr("src")
		if strings.Contains(js, ".js") {
			p.JSMap[js] = ""
		}
	})
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		img, _ := s.Attr("src")
		p.ImgMap[img] = ""
	})
	doc.Find("link").Each(func(_ int, s *goquery.Selection) {
		css, _ := s.Attr("href")
		if strings.Contains(css, "css") {
			p.CSSMap[css] = ""
		}
	})
	p.ParseDomain()
	p.UUID = base64.StdEncoding.EncodeToString([]byte(p.URL))
	p.SetPath(ctx.OutputPath)
}

// WriteHTML html書き出し
func (p *Page) WriteHTML() error {
	file, err := os.Create(ctx.OutputPath + "/" + p.UUID + "/index.html")
	if err != nil {
		return err
	}
	defer file.Close()

	text, _ := p.Doc.Html()
	file.Write(([]byte)(text))
	return nil
}

// FetchFiles img, css, jsダウンロード
func (p *Page) FetchFiles() {
	dirs := fmt.Sprintf("%s/%s", ctx.OutputPath, p.UUID)
	if err := os.MkdirAll(dirs+"/"+fileTypeCSS, 0777); err != nil {
		logrus.Fatal(err)
	}
	if err := os.MkdirAll(dirs+"/"+fileTypeIMG, 0777); err != nil {
		logrus.Fatal(err)
	}
	if err := os.MkdirAll(dirs+"/"+fileTypeJS, 0777); err != nil {
		logrus.Fatal(err)
	}
	p.FetchCSS()
	p.FetchIMG()
	p.FetchJS()
	p.RewriteDoc()
	p.WriteHTML()
}

// FetchCSS cssファイル取得
func (p *Page) FetchCSS() {
	for k := range p.CSSMap {
		if !strings.Contains(k, httpToken) && !strings.Contains(k, httpsToken) {
			fileURL := fmt.Sprintf("%s://%s/%s", p.DomainScheme, p.Domain, k)
			logrus.Info(fileURL)
			if err := p.DownloadFile(fileURL, fileTypeCSS, k); err != nil {
				logrus.Fatal(err)
			}
		} else {
			logrus.Info(k)
			if err := p.DownloadFile(k, fileTypeCSS, k); err != nil {
				logrus.Fatal(err)
			}
		}
	}
}

// FetchIMG imgファイル取得
func (p *Page) FetchIMG() {
	for k := range p.ImgMap {
		if !strings.Contains(k, httpToken) && !strings.Contains(k, httpsToken) {
			fileURL := fmt.Sprintf("%s://%s/%s", p.DomainScheme, p.Domain, k)
			logrus.Info(fileURL)
			if err := p.DownloadFile(fileURL, fileTypeIMG, k); err != nil {
				logrus.Fatal(err)
			}
		} else {
			logrus.Info(k)
			if err := p.DownloadFile(k, fileTypeIMG, k); err != nil {
				logrus.Fatal(err)
			}
		}
	}
}

// FetchJS jsファイル取得
func (p *Page) FetchJS() {
	for k := range p.JSMap {
		if !strings.Contains(k, httpToken) && !strings.Contains(k, httpsToken) {
			fileURL := fmt.Sprintf("%s://%s/%s", p.DomainScheme, p.Domain, k)
			logrus.Info(fileURL)
			if err := p.DownloadFile(fileURL, fileTypeJS, k); err != nil {
				logrus.Fatal(err)
			}
		} else {
			logrus.Info(k)
			if err := p.DownloadFile(k, fileTypeJS, k); err != nil {
				logrus.Fatal(err)
			}
		}
	}
}

// SetLevel レベル設定
func (p *Page) SetLevel(n int) {
	p.Level = n
}

// SetDoc goquery doc セット
func (p *Page) SetDoc(doc *goquery.Document) {
	p.Doc = doc
}

// SetPath ファイル出力時のパス設定
func (p *Page) SetPath(base string) {
	s := strings.Join([]string{base, p.UUID}, "/") + "/"
	p.Path = s
}

// ParseDomain ドメイン取得
func (p *Page) ParseDomain() {
	u, err := url.Parse(p.URL)
	if err != nil {
		return
	}
	p.Domain = u.Host
	p.DomainScheme = u.Scheme
}

// DownloadFile ファイルのダウンロード
func (p *Page) DownloadFile(url, t, hashKey string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	_, fileName := path.Split(url)
	switch t {
	case fileTypeCSS:
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", ctx.OutputPath, p.UUID, fileTypeCSS, fileName)
		writeFile(fileFullPath, body)
		// rewrite map
		abs, _ := filepath.Abs("./")
		p.CSSMap[hashKey] = strings.Join([]string{abs, fileFullPath}, "/")
	case fileTypeIMG:
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", ctx.OutputPath, p.UUID, fileTypeIMG, fileName)
		writeFile(fileFullPath, body)
		// rewrite map
		abs, _ := filepath.Abs("./")
		p.ImgMap[hashKey] = strings.Join([]string{abs, fileFullPath}, "/")
	case fileTypeJS:
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", ctx.OutputPath, p.UUID, fileTypeJS, fileName)
		writeFile(fileFullPath, body)
		// rewrite map
		abs, _ := filepath.Abs("./")
		p.JSMap[hashKey] = strings.Join([]string{abs, fileFullPath}, "/")
	default:
		break
	}
	return nil
}

// RewriteDoc img,css,js参照のためattrの書き換え
func (p *Page) RewriteDoc() {
	// TODO a href の書き換え
	p.Doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		js, _ := s.Attr("src")
		if strings.Contains(js, ".js") {
			s.SetAttr("src", p.JSMap[js])
		}
	})
	p.Doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		img, _ := s.Attr("src")
		s.SetAttr("src", p.ImgMap[img])
	})
	p.Doc.Find("link").Each(func(_ int, s *goquery.Selection) {
		css, _ := s.Attr("href")
		if strings.Contains(css, "css") {
			s.SetAttr("href", p.CSSMap[css])
		}
	})
}

func writeFile(filePath string, body []byte) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	file.Write(body)
	return nil
}
