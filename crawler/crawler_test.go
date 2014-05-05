package crawler_test

import (
	"fmt"
	"github.com/mcmillhj/gocrawl/crawler"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewCrawler_1(t *testing.T) {
	_, err := crawler.NewCrawler("123")
	if err == nil {
		t.Error("[TestNewCrawler_1]: Did not throw error on fake url")
	} else {
		t.Log("Passed TestNewCrawler_1")
	}
}

func TestNewCrawler_2(t *testing.T) {
	_, err := crawler.NewCrawler("http://www.google.com")
	if err != nil {
		t.Error("[TestNewCrawler_2]: Threw error on real url")
	} else {
		t.Log("Passed TestNewCrawler_2")
	}
}

func TestCrawl_1(t *testing.T) {
	// create a mock server to serve a prefetched HTML page
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, testHTMLAssets)
	}))
	defer mockServer.Close()

	crawler, _ := crawler.NewCrawler(mockServer.URL)
	pages, _ := crawler.Crawl()

	// got back the assets for this page
	// ignored the fragment
	if len(pages[0].Assets) != 2 {
		t.Error("[TestCrawl_1] Did not gather all assets")
	} else {
		t.Log("Passed TestCrawl_1")
	}
}

func TestCrawl_2(t *testing.T) {
	// create a mock server to serve a prefetched HTML page
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, testHTMLLinks)
	}))
	defer mockServer.Close()

	crawler, _ := crawler.NewCrawler(mockServer.URL)
	pages, _ := crawler.Crawl()

	// got back, the starting page and 5 pages for links
	if len(pages) != 5 {
		t.Error("[TestCrawl_2] Did not gather all links")
	} else {
		t.Log("Passed TestCrawl_2")
	}
}

var testHTMLAssets string = `
<html>
   <head>
      <link rel="stylesheet" type="text/css" href="/news.css?WYHpXE7l12wmoXJMMN3N">
      <link rel="shortcut icon" href="/favicon.ico">
      <a href="#">FRAGMENT</a>
      <a href="//schemetest.js">Schemetest</a>
   </head>
   <body>
   </body>
</html>`

var testHTMLLinks string = `
<html>
   <body>
      <a href="/yahoo">Yahoo</a>
      <a href="/google">Google</a>
      <a href="/bing">Bing</a>
      <a href="/duckduckgo">DuckDuckGo</a>
   </body>
</html>`
