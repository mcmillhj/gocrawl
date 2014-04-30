package crawler_test

import (
	"github.com/mcmillhj/gocrawl/crawler"
	"testing"
)

func TestNewCrawler_1(t *testing.T) {
	_, err := crawler.NewCrawler("123")
	if err == nil {
		t.Error("[TestNewCrawler_1]: Did not throw error on fake url")
	} else {
		t.Log("Passed TestNewCrawler_2")
	}
}
