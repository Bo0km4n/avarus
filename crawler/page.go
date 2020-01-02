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
	level        int
	doc          *goquery.Document
	fqdn         string
	domainSchema string
	path         string
	url          string
	uuid         string
	imgMap       map[string]string
	cssMap       map[string]string
	linkMap      map[string]string
	jsMap        map[string]string
	cssImgMap    map[string]string
}

// NewPage Page生成
func NewPage(url string, level int) Page {
	page := Page{
		level:     level,
		url:       url,
		imgMap:    map[string]string{},
		linkMap:   map[string]string{},
		jsMap:     map[string]string{},
		cssMap:    map[string]string{},
		cssImgMap: map[string]string{},
	}
	return page
}

// Exec ページクローリング実行
func (p *Page) Exec() {
	if _, isVisited := E.Refferer[p.url]; isVisited {
		return
	}
	p.start()
	p.queuingPages()
	p.fetchFiles()
	p.rewriteDoc()
	p.writeHTML()
	return
}

// Init 構造体初期化
func (p *Page) start() {
	doc, err := goquery.NewDocument(p.url)
	p.doc = doc
	if err != nil {
		logrus.Fatalf("Failed get document %s", p.url)
		return
	}
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		if !strings.Contains(link, ".img") && (strings.Contains(link, ".html") || strings.Contains(link, "http")) {
			p.linkMap[link] = ""
		}
	})
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		js, _ := s.Attr("src")
		if strings.Contains(js, ".js") {
			p.jsMap[js] = ""
		}
	})
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		img, _ := s.Attr("src")
		p.imgMap[img] = ""
	})
	doc.Find("link").Each(func(_ int, s *goquery.Selection) {
		rel, _ := s.Attr("rel")
		css, _ := s.Attr("href")
		if strings.Contains(rel, "stylesheet") {
			p.cssMap[css] = ""
		}
	})
	p.parseDomain()
	p.uuid = getPageName(p.url)
	p.SetPath(E.OutputPath)
}

// queuingPages リンクページを格納
func (p *Page) queuingPages() {
	// Link書き換え
	m := new(sync.Mutex)
	p.doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		var linkURL string
		link, _ := s.Attr("href")
		abs, _ := filepath.Abs("./")
		linkURL = p.getLinkURL(link)
		linkPath := getPageName(linkURL)

		p.linkMap[link] = strings.Join([]string{abs, E.OutputPath, linkPath, "index.html"}, "/")
		s.SetAttr("href", p.linkMap[link])

		// E.Pages に格納
		if p.level <= E.Depth {
			newPage := NewPage(linkURL, p.level+1)
			m.Lock()
			E.Pages = append(E.Pages, newPage)
			m.Unlock()
		}
	})
	E.Refferer[p.url] = ""
}

// writeHTML html書き出し
func (p *Page) writeHTML() error {
	file, err := os.Create(E.OutputPath + "/" + p.uuid + "/index.html")
	if err != nil {
		return err
	}
	defer file.Close()

	text, _ := p.doc.Html()
	file.Write(([]byte)(text))
	return nil
}

// fetchFiles img, css, jsダウンロード
func (p *Page) fetchFiles() {
	dirs := fmt.Sprintf("%s/%s", E.OutputPath, p.uuid)
	if err := os.MkdirAll(dirs+"/"+fileTypeCSS, 0777); err != nil {
		logrus.Fatal(err)
	}
	if err := os.MkdirAll(dirs+"/"+fileTypeIMG, 0777); err != nil {
		logrus.Fatal(err)
	}
	if err := os.MkdirAll(dirs+"/"+fileTypeJS, 0777); err != nil {
		logrus.Fatal(err)
	}
	p.fetchCSS()
	p.fetchIMG()
	p.fetchJS()
}

// FetchCSS cssファイル取得
func (p *Page) fetchCSS() {
	for k := range p.cssMap {
		if !strings.Contains(k, httpToken) && !strings.Contains(k, httpsToken) {
			fileURL := fmt.Sprintf("%s://%s/%s", p.domainSchema, p.fqdn, k)
			logrus.Infof("FQDN: %s, css: %s", p.fqdn, fileURL)
			if err := p.downloadFile(fileURL, fileTypeCSS, k); err != nil {
				logrus.Warn(err)
			}
		} else {
			logrus.Infof("FQDN: %s, css: %s", p.fqdn, k)
			if err := p.downloadFile(k, fileTypeCSS, k); err != nil {
				logrus.Warn(err)
			}
		}
	}
}

// FetchIMG imgファイル取得
func (p *Page) fetchIMG() {
	for k := range p.imgMap {
		if !strings.Contains(k, httpToken) && !strings.Contains(k, httpsToken) {
			fileURL := fmt.Sprintf("%s://%s/%s", p.domainSchema, p.fqdn, k)
			logrus.Infof("FQDN: %s, img: %s", p.fqdn, fileURL)
			if err := p.downloadFile(fileURL, fileTypeIMG, k); err != nil {
				logrus.Fatal(err)
			}
		} else {
			logrus.Infof("FQDN: %s, img: %s", p.fqdn, k)
			if err := p.downloadFile(k, fileTypeIMG, k); err != nil {
				logrus.Fatal(err)
			}
		}
	}
}

// FetchJS jsファイル取得
func (p *Page) fetchJS() {
	for k := range p.jsMap {
		if !strings.Contains(k, httpToken) && !strings.Contains(k, httpsToken) {
			fileURL := fmt.Sprintf("%s://%s/%s", p.domainSchema, p.fqdn, k)
			logrus.Infof("FQDN: %s, javascript: %s", p.fqdn, fileURL)
			if err := p.downloadFile(fileURL, fileTypeJS, k); err != nil {
				logrus.Fatal(err)
			}
		} else {
			logrus.Infof("FQDN: %s, javascript: %s", p.fqdn, k)
			if err := p.downloadFile(k, fileTypeJS, k); err != nil {
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
		return p.domainSchema + ":" + link
	case isStartWithRelative(link):
		return getAbsURLFromRelative(p, link)
	}
	return p.domainSchema + "://" + strings.Join([]string{p.fqdn, link}, "/")
}

// SetLevel レベル設定
func (p *Page) SetLevel(n int) {
	p.level = n
}

// SetDoc goquery doc セット
func (p *Page) SetDoc(doc *goquery.Document) {
	p.doc = doc
}

// SetPath ファイル出力時のパス設定
func (p *Page) SetPath(base string) {
	s := strings.Join([]string{base, p.uuid}, "/") + "/"
	p.path = s
}

// parseDomain ドメイン取得
func (p *Page) parseDomain() {
	u, err := url.Parse(p.url)
	if err != nil {
		return
	}
	p.fqdn = u.Host
	p.domainSchema = u.Scheme
}

// DownloadFile ファイルのダウンロード
func (p *Page) downloadFile(url, t, hashKey string) error {
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
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", E.OutputPath, p.uuid, fileTypeCSS, fileName)
		extractURL(p, string(body), url)
		css := replaceCSSImgText(p, string(body))
		bs := []byte(css)
		writeFile(fileFullPath, bs)
		// rewrite map
		abs, _ := filepath.Abs("./")
		p.cssMap[hashKey] = strings.Join([]string{abs, fileFullPath}, "/")
	case fileTypeIMG:
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", E.OutputPath, p.uuid, fileTypeIMG, fileName)
		writeFile(fileFullPath, body)
		// rewrite map
		abs, _ := filepath.Abs("./")
		p.imgMap[hashKey] = strings.Join([]string{abs, fileFullPath}, "/")
	case fileTypeJS:
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", E.OutputPath, p.uuid, fileTypeJS, fileName)
		writeFile(fileFullPath, body)
		// rewrite map
		abs, _ := filepath.Abs("./")
		p.jsMap[hashKey] = strings.Join([]string{abs, fileFullPath}, "/")
	default:
		break
	}
	return nil
}

// rewriteDoc img,css,js参照のためattrの書き換え
func (p *Page) rewriteDoc() {
	p.doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		js, _ := s.Attr("src")
		if strings.Contains(js, ".js") {
			s.SetAttr("src", p.jsMap[js])
		}
	})
	p.doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		img, _ := s.Attr("src")
		s.SetAttr("src", p.imgMap[img])
	})
	p.doc.Find("link").Each(func(_ int, s *goquery.Selection) {
		rel, _ := s.Attr("rel")
		css, _ := s.Attr("href")
		if strings.Contains(rel, "stylesheet") {
			s.SetAttr("href", p.cssMap[css])
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
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s%s", E.OutputPath, p.uuid, fileTypeIMG, uuid.NewV4().String(), ext)
		abs, _ := filepath.Abs("./")
		downloadURL := ""
		if isStartWithRelative(url) {
			orgURL := url
			relativePath, _ := path.Split(cssURL)
			relativePath = strings.Replace(relativePath, p.domainSchema+"://", "", 1)
			pathArray := strings.Split(relativePath, "/")
			relativePath = p.domainSchema + "://" + strings.Join(pathArray[:len(pathArray)-2], "/")
			downloadURL = strings.Replace(url, "../", "", 1)
			downloadURL = strings.Join([]string{relativePath, downloadURL}, "/")
			p.cssImgMap[orgURL] = strings.Join([]string{abs, fileFullPath}, "/")
		} else if isStartWithCurrentPath(url) {
			orgURL := url
			paths := strings.Split(p.url, "/")
			downloadURL = p.domainSchema + "://" + strings.Join(paths[:len(paths)-1], "/")
			downloadURL = downloadURL + "/" + strings.Replace(url, currentPathToken, "", 1)
			p.cssImgMap[orgURL] = strings.Join([]string{abs, fileFullPath}, "/")
		} else {
			if !strings.Contains(url, httpToken) && !strings.Contains(url, httpsToken) {
				downloadURL = p.domainSchema + "://" + p.fqdn + "/" + url
				p.cssImgMap[url] = strings.Join([]string{abs, fileFullPath}, "/")
			} else {
				downloadURL = url
				p.cssImgMap[url] = strings.Join([]string{abs, fileFullPath}, "/")
			}
		}
		downloadFileInCSS(p, downloadURL, fileFullPath)
	}
}

// css内の画像ファイルをダウンロード
func downloadFileInCSS(p *Page, url string, savePath string) {
	logrus.Infof("FQDN: %s, file_in_css: %s", p.fqdn, url)
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
	for k, v := range p.cssImgMap {
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

// p.url = http://www.google.co.jp/hoge/index.html
// link = ../../hello.jpg
// return http://www/google.co.jp/hello.jpg
func getAbsURLFromRelative(p *Page, link string) string {
	uri := strings.Replace(p.url, "http://", "", -1)
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
	return p.domainSchema + "://" + strings.Join([]string{domain, link}, "/")

}
