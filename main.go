package main

import (
	"github.com/mcmillhj/gocrawl/crawler"
)

func main() {
	c, err := crawler.NewCrawler("http://www.digitalocean.com")
	if err != nil {
		os.Exit(1)
	}
	c.Crawl()
}
