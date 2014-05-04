package main

import (
	"fmt"
	"github.com/mcmillhj/gocrawl/crawler"
	"os"
)

func main() {
	c, err := crawler.NewCrawler("http://www.digitalocean.com")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pages, err := c.Crawl()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// print out all Pages
	urlCount := 0
	for _, page := range pages {
		fmt.Println("Page: ", page.Referrer)
		fmt.Println("Urls: ")
		for i, u := range page.Urls {
			fmt.Printf("\t%d -> %s\n", i, u)
			urlCount++
		}
		fmt.Println("Assets: ")
		for j, a := range page.Assets {
			fmt.Printf("\t%d -> %s\n", j, a)
		}
	}

	fmt.Println("Crawled", len(pages), "pages with", urlCount, "unique urls")
}
