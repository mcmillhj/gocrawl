package crawler

import (
	"code.google.com/p/go.net/html"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

var (
	INFO  *log.Logger
	ERROR *log.Logger
)

func init() {
	INFO = log.New(os.Stdout, "[INFO] ", log.Ltime)
	ERROR = log.New(os.Stderr, "[ERROR] ", log.Ltime)
}

type Page struct {
	referrer string
	urls     []string
	assets   []string
}

type Crawler struct {
	startUrl string
	host     string
	visited  map[string]bool
	sitemap  map[string]Page
}

func NewCrawler(startUrl string) *Crawler {
	u, err := url.Parse(startUrl)
	if err != nil {
		ERROR.Println(err)
		os.Exit(1)
	}
	return &Crawler{startUrl, u.Host, make(map[string]bool), make(map[string]Page)}
}

func (crawler *Crawler) Crawl() {
	crawler.crawl(crawler.startUrl)
}

func (crawler *Crawler) crawl(url string) {
	response, err := http.Get(url)
	if err != nil {
		ERROR.Println(err)
		os.Exit(1)
	}

	defer response.Body.Close()
	tree, err := html.Parse(response.Body)
	if err != nil {
		ERROR.Println(err)
		os.Exit(1)
	}
	page := &Page{url, make([]string, 0), make([]string, 0)}
	crawler.visited[url] = true
	links := crawler.gatherLinks(tree, url)
	// assets := crawler.gatherAssets(tree)
	fmt.Println(page, links)
}

func (crawler *Crawler) gatherAssets(n *html.Node) []string {
	// only examine img, link, or script tags
	if n.Type == html.ElementNode &&
		(n.Data == "img" || n.Data == "link" || n.Data == "script") {
		for _, img := range n.Attr {
			switch img.Key {
			case "src":
				fmt.Println(img.Val)
			case "href":
				fmt.Println(img.Val)
			}
		}
	}

	// recurse over other nodes in the HTML tree
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		crawler.gatherAssets(c)
	}

	return []string{"-1"}
}

func (crawler *Crawler) gatherLinks(n *html.Node, ref string) []string {
	urlParts := regexp.MustCompile(`\.`).Split(crawler.host, 3)
	domainPattern := fmt.Sprintf("^(?:https?://)?(?:www\\.)?%s\\.%s.*$", urlParts[1], urlParts[2])
	domainRegex := regexp.MustCompile(domainPattern)

	// only examine links
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				u, err := url.Parse(a.Val)
				// if a url is invalid, it cannot be crawled. skip it.
				if err != nil {
					INFO.Println(err)
					continue
				}

				// handle relative urls by prepending the domain
				if !u.IsAbs() {
					// ignore fragments
					if u.Path == "" {
						INFO.Println("Skipped URL, fragment")
						continue
					}
					// prepend referring page
					INFO.Println("Found relative URL ", u.String(), " prepending referrer")
					u.Path = ref + u.Path
				}
				if !domainRegex.MatchString(u.String()) {
					INFO.Println("Skipped URL ", u.String(), " not of domain ", crawler.host)
					continue
				}
				// if we have already seen this url, do not record it again
				if crawler.visited[u.String()] {
					INFO.Println("Skipped URL ", u.String(), " have already seen it")
					continue
				}
				crawler.visited[u.String()] = true
				fmt.Println(u.String())
			}
		}
	}

	// recurse over other nodes in the HTML tree
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		crawler.gatherLinks(c, ref)
	}

	return []string{"-1"}
}
