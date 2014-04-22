package main

import (
	"github.com/mcmillhj/gocrawl/crawler"
)

func main() {
	c := crawler.NewCrawler("http://www.digitalocean.com")
	c.Crawl()
}
