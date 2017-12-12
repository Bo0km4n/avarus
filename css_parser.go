package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// ExtractURL 文字列からurl("http://****") を抽出
func (p *Page) ExtractURL(line string) {
	re := regexp.MustCompile(`url\(\"http.*?\"\)`)
	result := re.FindAllStringSubmatch(line, -1)
	for _, v := range result {
		url := strings.Replace(v[0], "url(\"", "", 1)
		url = strings.Replace(url, "\")", "", 1)
		_, fileName := path.Split(url)
		fileFullPath := fmt.Sprintf("%s/%s/%s/%s", ctx.OutputPath, p.UUID, fileTypeIMG, fileName)
		abs, _ := filepath.Abs("./")
		p.CSSImgMap[url] = strings.Join([]string{abs, fileFullPath}, "/")
		downloadFileInCSS(p, url)
	}
}

func downloadFileInCSS(p *Page, url string) {
	logrus.Info(url)
	response, err := http.Get(url)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	_, fileName := path.Split(url)
	fileFullPath := fmt.Sprintf("%s/%s/%s/%s", ctx.OutputPath, p.UUID, fileTypeIMG, fileName)
	writeFile(fileFullPath, body)
}

func replaceCSSImg(p *Page, text string) string {
	for k, v := range p.CSSImgMap {
		text = strings.Replace(text, k, v, 1)
	}
	return text
}
