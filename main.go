package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

var wikiPrefix = "https://en.wikipedia.org/wiki/"
var cacheDir = "pageCache"

func main() {
	fmt.Println("start crawly")

	var wg sync.WaitGroup
	wg.Add(3)

	go loadPagesFromPage("George_W._Bush", &wg)
	go loadPagesFromPage("Roman_Empire", &wg)
	go loadPagesFromPage("Napoleon", &wg)
	wg.Wait()
}

func loadPagesFromPage(pageName string, wg *sync.WaitGroup) {
	startPage := getPage(pageName)
	links := getLinks(startPage)
	filtered := filterLinks(links)

	err := os.MkdirAll(cacheDir, 0755)
	if err != nil {
		log.Fatal(err)
	}

	for pageName := range filtered {
		getPage(pageName)
	}

	wg.Done()
}

func getPage(pageName string) string {
	hit, pageStr := checkCache(pageName)
	if hit {
		fmt.Printf("Cache hit on %s\n", pageName)
		return pageStr
	}

	page := requestPageFromWiki(pageName)
	writeToCache(pageName, []byte(page))
	return page
}

func writeToCache(pageName string, data []byte) {
	fmt.Printf("Writing %s to cache\n", pageName)
	os.WriteFile("./"+cacheDir+"/"+pageName, data, 0644)
}

func checkCache(pageName string) (hit bool, page string) {
	filePath := "./" + cacheDir + "/" + pageName
	_, err := os.Stat(filePath)
	if err == nil {
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}
		return true, string(bytes)
	}
	return false, ""
}

func requestPageFromWiki(pageName string) string {
	req, err := http.NewRequest("GET", wikiPrefix+pageName, nil)
	if err != nil {
		log.Fatal(err)
	}

	userAgent := "Crawly/0.1 (ariel.itskovich@gmail.com)"
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode == 429 {
		log.Fatal("Got 429")
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes)
}

func getLinks(htmlStr string) map[string]bool {
	root, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		log.Fatal(err)
	}

	links := make(map[string]bool)
	traverseNode(root, &links)
	return links
}

// returns found links
func traverseNode(n *html.Node, links *map[string]bool) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				(*links)[attr.Val] = true
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		traverseNode(c, links)
	}
}

func filterLinks(links map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for link := range links {
		cut, hadPrefix := strings.CutPrefix(link, wikiPrefix)
		if hadPrefix && !strings.Contains(cut, ":") && cut[0] != '\'' && cut[0] != '"' {
			parts := strings.Split(cut, "#")
			result[parts[0]] = true
		}
	}
	return result
}
