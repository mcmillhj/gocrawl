package crawler

import (
	"code.google.com/p/go.net/html" // HTML parser
	"fmt"
	"github.com/PuerkitoBio/purell" // URL Normalizer
	"io/ioutil"
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
	referrer string
	urls     []string
	assets   []string
}

type Crawler struct {
	startUrl    string
	host        string
	domainRegex *regexp.Regexp
	doNotCrawl  map[string]bool
	visited     map[string]bool
	errors      map[string]bool
	sitemap     map[string]*Page
}

func NewCrawler(startUrl string) *Crawler {
	u, err := url.Parse(startUrl)
	if err != nil {
		ERROR.Println(err)
		os.Exit(1)
	}

	// build domain regex
	urlParts := regexp.MustCompile(`\.`).Split(u.Host, 3)
	// break hostname into pieces www.google.com -> ['www', 'google', 'com']
	domainPattern := fmt.Sprintf("^(?:https?://)?(?:www\\.)?%s\\.%s.*$", urlParts[1], urlParts[2])
	domainRegex := regexp.MustCompile(domainPattern)

	return &Crawler{
		startUrl,
		u.Host,
		domainRegex,
		nil,
		make(map[string]bool),
		make(map[string]bool),
		make(map[string]*Page),
	}
}

func (crawler *Crawler) Crawl() {
	// download robots.txt and parse
	crawler.doNotCrawl = crawler.parseRobots()

	// urls to crawl
	urls := []string{crawler.startUrl}

	for i := 0; i < len(urls); i++ {
		u := urls[i]

		// skip urls we have already crawled
		// or already recorded an error on
		// or robots.txt says not to crawl this page
		if crawler.visited[u] || crawler.errors[u] || crawler.doNotCrawl[u] {
			continue
		}
		// INFO.Println("Crawling", u)
		crawledPage := crawler.crawl(u)
		if crawledPage == nil {
			continue
		}

		// save this Page
		crawler.sitemap[u] = crawledPage

		// mark this url as visited
		crawler.visited[u] = true

		// add any urls returned from this Page to the queue to be crawled
		urls = append(urls, crawledPage.urls...)
	}

	var urlcount int = 0
	// print out all Pages
	for url, page := range crawler.sitemap {
		fmt.Println("Page: ", url)
		fmt.Println("Urls  : ")
		for i, u := range page.urls {
			urlcount++
			fmt.Printf("\t%d -> %s\n", i, u)
		}
		fmt.Println("Assets: ")
		for i, a := range page.assets {
			fmt.Printf("\t%d -> %s\n", i, a)
		}
	}

	fmt.Println("Crawled", len(crawler.sitemap), "pages, with a total of", urlcount, "urls")
}

func (crawler *Crawler) crawl(url string) *Page {
	response, err := http.Get(url)
	if err != nil {
		// ERROR.Println(err)
		// os.Exit(1)
		return nil
	}

	if !strings.Contains(response.Header["Content-Type"][0], "text/html") {
		// INFO.Println("HTTP GET", url, "has Content-Type of", response.Header["Content-Type"])
		crawler.errors[url] = true
		return nil
	}

	// only proceed to pages with 200 response code
	if response.StatusCode != 200 {
		// INFO.Println("HTTP GET", url, "returned status:", response.Status)
		crawler.errors[url] = true
		return nil
	}

	// ignore pages that redirect to a non domain page
	if !crawler.domainRegex.MatchString(response.Request.URL.String()) {
		// INFO.Println("Redirect from", url, "to non-domain url", response.Request.URL.String(), "detected")
		crawler.errors[url] = true
		return nil
	}

	defer response.Body.Close()
	tree, err := html.Parse(response.Body)
	if err != nil {
		// ERROR.Println(err)
		os.Exit(1)
	}

	// create a new page to represent this url
	return &Page{
		response.Request.URL.String(),
		crawler.gatherLinks(tree, url),
		crawler.gatherAssets(tree, url),
	}
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
				assets = append(assets, crawler.processUrl(img.Val, ref))
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
				u := crawler.processUrl(a.Val, ref)

				// errors handled by processUrl
				if u == "" {
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

func (crawler *Crawler) processUrl(href, ref string) string {

	// normalize url
	urlString, err := purell.NormalizeURLString(href, purell.FlagsSafe|purell.FlagsUsuallySafeNonGreedy)
	if err != nil {
		// INFO.Println(err)
		return ""
	}

	u, err := url.Parse(urlString)
	// if a url is invalid, it cannot be crawled. skip it.
	if err != nil {
		// INFO.Println(err)
		return ""
	}

	// handle relative urls by prepending the domain
	if !u.IsAbs() {
		// ignore fragments
		if u.Path == "" {
			// INFO.Println("Skipped URL, fragment")
			return ""
		}

		// process links with // prepended, which means inherit the current page's protocol
		if u.Host != "" {
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

	return u.String()
}

func (crawler *Crawler) parseRobots() map[string]bool {
	response, err := http.Get(crawler.startUrl + "/robots.txt")
	if err != nil {
		// handle error
	}
	defer response.Body.Close()
	robots, err := ioutil.ReadAll(response.Body)

	doNotCrawlMap := make(map[string]bool)
	disallowPattern := regexp.MustCompile("Disallow: (.*)$")
	for _, line := range strings.Split(string(robots), "\n") {
		if matches := disallowPattern.FindStringSubmatch(line); len(matches) > 0 {
			doNotCrawlMap[crawler.processUrl(matches[1], crawler.startUrl)] = true
		}
	}

	return doNotCrawlMap
}
