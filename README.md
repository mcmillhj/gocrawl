This is my first time using the Go programming language, so I thought I would
write a simple web crawler.

The crawler will only accept a single domain, and gather a mapping of the site
and all assets of each page.

Overview: 
1. Accepts a single domain
2. Does not crawl subdomains
3. Obeys robots.txt (if one can be found)
4. Examines Content-Type header of the http.Get response, discards anything
   with a 'Content-Type' that is not 'text/html' 

Future Work: 

1. re-factor crawl into a goroutine so more than one crawl can be happening at
   a time
2. obey robots.txt
3. research more idiomatic testing practices in Go
4. refactor Page into its own package

