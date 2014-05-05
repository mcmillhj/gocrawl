package crawler

import (
	"code.google.com/p/go.net/html"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
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
	Referrer string
	Urls     []string
	Assets   []string
}

type Crawler struct {
	startUrl    string
	host        string
	domainRegex *regexp.Regexp
	visited     map[string]bool
}

func NewCrawler(startUrl string) (*Crawler, error) {
	u, err := url.Parse(startUrl)
	if err != nil {
		ERROR.Println(err)
		return nil, err
	}

	if u.Host == "" {
		return nil, errors.New("Invalid starting URL" + startUrl)
	}

	// build domain regex
	// test for local servers or IP addresses
	var domainRegex *regexp.Regexp
	if strings.Count(u.Host, ".") > 2 {
		domainRegex = regexp.MustCompile(".*")
	} else {
		urlParts := regexp.MustCompile(`\.`).Split(u.Host, 3)

		// break hostname into pieces www.google.com -> ['www', 'google', 'com']
		domainPattern := fmt.Sprintf("^(?:https?://)?(?:www\\.)?%s\\.%s.*$", urlParts[1], urlParts[2])
		domainRegex = regexp.MustCompile(domainPattern)
	}

	return &Crawler{
		startUrl,
		u.Host,
		domainRegex,
		make(map[string]bool),
	}, nil
}

func (crawler *Crawler) Crawl() ([]*Page, error) {
	// urls to crawl
	urls := []string{crawler.startUrl}

	// pages already crawled
	pages := make([]*Page, 0)

	// errors encountered to avoid recrawling
	errors := make(map[string]bool)

	for i := 0; i < len(urls); i++ {
		u := urls[i]

		// skip urls we have already crawled
		if crawler.visited[u] || errors[u] {
			continue
		}

		INFO.Println("Crawling", u)
		crawledPage, err := crawler.crawl(u)
		if err != nil {
			errors[u] = true
			continue
		}

		// save this Page
		pages = append(pages, crawledPage)

		// mark this url as visited
		crawler.visited[u] = true

		// add any urls returned from this Page to the queue to be crawled
		urls = append(urls, crawledPage.Urls...)
	}

	return pages, nil
}

func (crawler *Crawler) crawl(url string) (*Page, error) {
	response, err := http.Get(url)
	if err != nil {
		ERROR.Println(err)
		return nil, err
	}

	// only crawl HTML pages
	if !strings.Contains(response.Header["Content-Type"][0], "text/html") {
		INFO.Println("HTTP GET", url, "has Content-Type of", response.Header["Content-Type"])
		return nil, errors.New("LOG: Invalid Content-Type")
	}

	// only proceed to pages with 200 response code
	if response.StatusCode != 200 {
		INFO.Println("HTTP GET", url, "returned status:", response.Status)
		return nil, errors.New("")
	}

	// detect redirect to subdomain or other domain
	if !crawler.domainRegex.MatchString(response.Request.URL.String()) {
		INFO.Println("HTTP GET", url, "detected redirect to", response.Request.URL.String())
		return nil, errors.New("")
	}

	defer response.Body.Close()
	tree, err := html.Parse(response.Body)
	if err != nil {
		ERROR.Println(err)
		return nil, err
	}

	// create a new page to represent this url
	return &Page{
		url,
		crawler.gatherLinks(tree, url),
		crawler.gatherAssets(tree, url),
	}, nil
}

func (crawler *Crawler) gatherAssets(n *html.Node, ref string) []string {
	// create a slice to hold assets
	assets := make([]string, 0)

	// only examine img, link, or script tags
	if n.Type == html.ElementNode &&
		(n.Data == "img" || n.Data == "link" || n.Data == "script") {
		for _, img := range n.Attr {
			switch img.Key {
			case "src", "href":
				u, err := crawler.processUrl(img.Val, ref)
				if err == nil {
					assets = append(assets, u)
				}
			}
		}
	}

	// recurse over other nodes in the HTML tree
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		newAssets := crawler.gatherAssets(c, ref)
		if newAssets != nil {
			assets = append(assets, newAssets...)
		}
	}

	return assets
}

func (crawler *Crawler) gatherLinks(n *html.Node, ref string) []string {
	// create a slice to hold urls
	links := make([]string, 0)

	// only examine links
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				u, err := crawler.processUrl(a.Val, ref)

				// errors handled by processUrl
				if err != nil {
					continue
				}

				// ignore links not of this domain
				if !crawler.domainRegex.MatchString(u) {
					// INFO.Println("Skipped URL", u, "not of domain", crawler.host)
					continue
				}

				// add it to the list of urls found on this page
				links = append(links, u)
			}
		}
	}

	// recurse over other HTML nodes on this page gathering links
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		newLinks := crawler.gatherLinks(c, ref)
		if newLinks != nil {
			links = append(links, newLinks...)
		}
	}

	return links
}

func (crawler *Crawler) processUrl(href, ref string) (string, error) {
	u, err := url.Parse(href)
	// if a url is invalid, it cannot be crawled. skip it.
	if err != nil {
		// INFO.Println(err)
		return "", err
	}

	// handle relative urls by prepending the domain
	if !u.IsAbs() {
		// ignore fragments
		if u.Path == "" {
			// INFO.Println("Skipped URL, fragment")
			return "", errors.New("")
		}

		// process links with // prepended, which means inherit the current page's protocol
		if u.Host != "" {
			fmt.Println(u)
			refURL, _ := url.Parse(ref)
			// INFO.Println("Found relative URL with preceding double slash", u.String(), "inheriting referrer page's protocol", refURL.Scheme)
			// prepend protocol of referring page to URL
			u.Scheme = refURL.Scheme
		} else {
			if !strings.HasPrefix(u.Path, "/") {
				// prepend referring page
				// INFO.Println("Found relative URL", u.String(), "prepending referrer", ref)
				u.Path = ref + u.Path
			} else {
				// prepend top-level domain
				// INFO.Println("Found relative URL", u.String(), "prepending domain", crawler.startUrl)
				u.Path = crawler.startUrl + u.Path
			}
		}
	}

	return u.String(), nil
}
