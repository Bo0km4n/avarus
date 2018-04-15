package crawler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const (
	httpToken        = "http://"
	httpsToken       = "https://"
	relativeToken    = "../"
	currentPathToken = "./"
	doubleSlashToken = "//"
	fileTypeCSS      = "css"
	fileTypeIMG      = "img"
	fileTypeJS       = "js"
)

// Page htmlページ構造体
type Page struct {
	Level        int
	Doc          *goquery.Document
	FQDN         string
	DomainScheme string
	Path         string
	URL          string
	UUID         string
	HTML         string
	ImgMap       map[string]string
	CSSMap       map[string]string
	LinkMap      map[string]string
	JSMap        map[string]string
	CSSImgMap    map[string]string
}

// NewPage Page生成
func NewPage(url string, level int) Page {
	page := Page{
		Level:     level,
		URL:       url,
		HTML:      "",
		ImgMap:    map[string]string{},
		LinkMap:   map[string]string{},
		JSMap:     map[string]string{},
		CSSMap:    map[string]string{},
		CSSImgMap: map[string]string{},
	}
	return page
}

// Exec ページクローリング実行
func (p *Page) Exec() {
	if _, isVisited := Ctx.Refferer[p.URL]; isVisited {
		return
	}
	p.Init()
	p.QueuingPages()
	p.FetchFiles()
	p.RewriteDoc()
	p.WriteHTML()
	return
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
		rel, _ := s.Attr("rel")
		css, _ := s.Attr("href")
		if strings.Contains(rel, "stylesheet") {
			p.CSSMap[css] = ""
		}
	})
	p.ParseDomain()
	p.UUID = getPageName(p.URL)
	p.SetPath(Ctx.OutputPath)
}

// QueuingPages リンクページを格納
func (p *Page) QueuingPages() {
	// Link書き換え
	m := new(sync.Mutex)
	p.Doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		var linkURL string
		link, _ := s.Attr("href")
		abs, _ := filepath.Abs("./")
		linkURL = p.getLinkURL(link)
		linkPath := getPageName(linkURL)

		p.LinkMap[link] = strings.Join([]string{abs, Ctx.OutputPath, linkPath, "index.html"}, "/")
		s.SetAttr("href", p.LinkMap[link])

		// Ctx.Pages に格納
		if p.Level <= Ctx.Depth {
			newPage := NewPage(linkURL, p.Level+1)
			m.Lock()
			Ctx.Pages = append(Ctx.Pages, newPage)
			m.Unlock()
		}
	})
	Ctx.Refferer[p.URL] = ""
}

// WriteHTML html書き出し
func (p *Page) WriteHTML() error {
	file, err := os.Create(Ctx.OutputPath + "/" + p.UUID + "/index.html")
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
	dirs := fmt.Sprintf("%s/%s", Ctx.OutputPath, p.UUID)
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
}

// FetchCSS cssファイル取得
func (p *Page) FetchCSS() {
	for k := range p.CSSMap {
		if !strings.Contains(k, httpToken) && !strings.Contains(k, httpsToken) {
			fileURL := fmt.Sprintf("%s://%s/%s", p.DomainScheme, p.FQDN, k)
			logrus.Infof("FQDN: %s, css: %s", p.FQDN, fileURL)
			if err := p.DownloadFile(fileURL, fileTypeCSS, k); err != nil {
				logrus.Warn(err)
			}
		} else {
			logrus.Infof("FQDN: %s, css: %s", p.FQDN, k)
			if err := p.DownloadFile(k, fileTypeCSS, k); err != nil {
				logrus.Warn(err)
			}
		}
	}
}

// FetchIMG imgファイル取得
func (p *Page) FetchIMG() {
	for k := range p.ImgMap {
		if !strings.Contains(k, httpToken) && !strings.Contains(k, httpsToken) {
			fileURL := fmt.Sprintf("%s://%s/%s", p.DomainScheme, p.FQDN, k)
			logrus.Infof("FQDN: %s, img: %s", p.FQDN, fileURL)
			if err := p.DownloadFile(fileURL, fileTypeIMG, k); err != nil {
				logrus.Fatal(err)
			}
		} else {
			logrus.Infof("FQDN: %s, img: %s", p.FQDN, k)
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
			fileURL := fmt.Sprintf("%s://%s/%s", p.DomainScheme, p.FQDN, k)
			logrus.Infof("FQDN: %s, javascript: %s", p.FQDN, fileURL)
			if err := p.DownloadFile(fileURL, fileTypeJS, k); err != nil {
				logrus.Fatal(err)
			}
		} else {
			logrus.Infof("FQDN: %s, javascript: %s", p.FQDN, k)
			if err := p.DownloadFile(k, fileTypeJS, k); err != nil {
				logrus.Fatal(err)
			}
		}
	}
}

// <a href>タグのリンクパスをURLに整形
func (p *Page) getLinkURL(link string) string {
	switch {
	case isStartWithHTTP(link):
		return link
	case isStartWithHTTPS(link):
		return link
	case isStartWithDoubleSlash(link):
		return p.DomainScheme + ":" + link
	case isStartWithRelative(link):
		return getAbsURLFromRelative(p, link)
	}
	return p.DomainScheme + "://" + strings.Join([]string{p.FQDN, link}, "/")
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
	p.FQDN = u.Host
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
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", Ctx.OutputPath, p.UUID, fileTypeCSS, fileName)
		extractURL(p, string(body), url)
		css := replaceCSSImgText(p, string(body))
		bs := []byte(css)
		writeFile(fileFullPath, bs)
		// rewrite map
		abs, _ := filepath.Abs("./")
		p.CSSMap[hashKey] = strings.Join([]string{abs, fileFullPath}, "/")
	case fileTypeIMG:
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", Ctx.OutputPath, p.UUID, fileTypeIMG, fileName)
		writeFile(fileFullPath, body)
		// rewrite map
		abs, _ := filepath.Abs("./")
		p.ImgMap[hashKey] = strings.Join([]string{abs, fileFullPath}, "/")
	case fileTypeJS:
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", Ctx.OutputPath, p.UUID, fileTypeJS, fileName)
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
		rel, _ := s.Attr("rel")
		css, _ := s.Attr("href")
		if strings.Contains(rel, "stylesheet") {
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

// https://www.google.com => return wwwgooglecom
func getPageName(url string) string {
	name := strings.Replace(url, ".", "", -1)
	name = strings.Replace(name, httpsToken, "", -1)
	name = strings.Replace(name, httpToken, "", -1)
	name = strings.Replace(name, "/", "", -1)
	name = strings.Replace(name, ":", "", -1)
	return name
}

// extractURL 文字列からurl("http://****") を抽出
func extractURL(p *Page, line, cssURL string) {
	re := regexp.MustCompile(`url\(.*?\)`)
	result := re.FindAllStringSubmatch(line, -1)
	for _, v := range result {
		url := replaceCSSImgURL(v[0])
		ext := filepath.Ext(url)
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s%s", Ctx.OutputPath, p.UUID, fileTypeIMG, uuid.NewV4().String(), ext)
		abs, _ := filepath.Abs("./")
		downloadURL := ""
		if isStartWithRelative(url) {
			orgURL := url
			relativePath, _ := path.Split(cssURL)
			relativePath = strings.Replace(relativePath, p.DomainScheme+"://", "", 1)
			pathArray := strings.Split(relativePath, "/")
			relativePath = p.DomainScheme + "://" + strings.Join(pathArray[:len(pathArray)-2], "/")
			downloadURL = strings.Replace(url, "../", "", 1)
			downloadURL = strings.Join([]string{relativePath, downloadURL}, "/")
			p.CSSImgMap[orgURL] = strings.Join([]string{abs, fileFullPath}, "/")
		} else if isStartWithCurrentPath(url) {
			orgURL := url
			paths := strings.Split(p.URL, "/")
			downloadURL = p.DomainScheme + "://" + strings.Join(paths[:len(paths)-1], "/")
			downloadURL = downloadURL + "/" + strings.Replace(url, currentPathToken, "", 1)
			p.CSSImgMap[orgURL] = strings.Join([]string{abs, fileFullPath}, "/")
		} else {
			if !strings.Contains(url, httpToken) && !strings.Contains(url, httpsToken) {
				downloadURL = p.DomainScheme + "://" + p.FQDN + "/" + url
				p.CSSImgMap[url] = strings.Join([]string{abs, fileFullPath}, "/")
			} else {
				downloadURL = url
				p.CSSImgMap[url] = strings.Join([]string{abs, fileFullPath}, "/")
			}
		}
		downloadFileInCSS(p, downloadURL, fileFullPath)
	}
}

// css内の画像ファイルをダウンロード
func downloadFileInCSS(p *Page, url string, savePath string) {
	logrus.Infof("FQDN: %s, file_in_css: %s", p.FQDN, url)
	response, err := http.Get(url)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	writeFile(savePath, body)
}

// css内のファイルパスをローカルのパスに書き換え
func replaceCSSImgText(p *Page, text string) string {
	for k, v := range p.CSSImgMap {
		text = strings.Replace(text, k, v, 1)
	}
	return text
}

func replaceCSSImgURL(url string) string {
	url = strings.Replace(url, "url(", "", 1)
	url = strings.Replace(url, ")", "", 1)
	url = strings.Replace(url, "\"", "", -1)
	url = strings.Replace(url, "'", "", -1)
	return url
}

// p.URL = http://www.google.co.jp/hoge/index.html
// link = ../../hello.jpg
// return http://www/google.co.jp/hello.jpg
func getAbsURLFromRelative(p *Page, link string) string {
	uri := strings.Replace(p.URL, "http://", "", -1)
	uri = strings.Replace(uri, "https://", "", -1)
	paths := strings.Split(uri, "/")

	relativeCount := strings.Count(link, "../")
	fmt.Println(uri, len(paths), paths, relativeCount)
	link = strings.Replace(link, "../", "", -1)

	var domain string
	if len(paths) < relativeCount {
		domain = paths[0]
	} else {
		domain = strings.Join(paths[:len(paths)-relativeCount], "/")
	}
	return p.DomainScheme + "://" + strings.Join([]string{domain, link}, "/")

}
