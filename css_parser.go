package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

// ExtractURL 文字列からurl("http://****") を抽出
func (p *Page) ExtractURL(line, cssURL string) {
	re := regexp.MustCompile(`url\(.*?\)`)
	result := re.FindAllStringSubmatch(line, -1)
	for _, v := range result {
		url := replaceCSSImgURL(v[0])
		ext := filepath.Ext(url)
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s%s", ctx.OutputPath, p.UUID, fileTypeIMG, uuid.NewV4().String(), ext)
		abs, _ := filepath.Abs("./")
		if strings.Contains(url, "../") {
			orgURL := url
			relativePath, _ := path.Split(cssURL)
			relativePath = strings.Replace(relativePath, p.DomainScheme+"://", "", 1)
			pathArray := strings.Split(relativePath, "/")
			relativePath = p.DomainScheme + "://" + strings.Join(pathArray[:len(pathArray)-2], "/")
			url = strings.Replace(url, "../", "", 1)
			url = strings.Join([]string{relativePath, url}, "/")
			p.CSSImgMap[orgURL] = strings.Join([]string{abs, fileFullPath}, "/")
		} else {
			p.CSSImgMap[url] = strings.Join([]string{abs, fileFullPath}, "/")
		}
		downloadFileInCSS(p, url, fileFullPath)
	}
}

func downloadFileInCSS(p *Page, url string, savePath string) {
	logrus.Info(url)
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

func replaceCSSImg(p *Page, text string) string {
	for k, v := range p.CSSImgMap {
		text = strings.Replace(text, k, v, 1)
	}
	return text
}

func replaceCSSImgURL(url string) string {
	url = strings.Replace(url, "url(", "", 1)
	url = strings.Replace(url, ")", "", 1)
	url = strings.Replace(url, "\"", "", 1)
	url = strings.Replace(url, "\"", "", 1)
	url = strings.Replace(url, "'", "", 1)
	url = strings.Replace(url, "'", "", 1)
	return url
}
