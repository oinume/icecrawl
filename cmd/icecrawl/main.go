package main

import (
	"os"

	"github.com/oinume/icecrawl/cli"
)

/*
See: https://www.firecrawl.dev/playground?url=https%3A%2F%2Fjournal.lampetty.net%2F&mode=crawl&limit=5&excludes=&includes=&formats=markdown&onlyMainContent=true&excludeTags=&includeTags=&includeSubdomains=true&mapSearch=&sessionId=this_is_just_a_preview_token
Input
- icecrawl scrape [options] <URL>
- icecrawl crawl [options] <URL>
*/
func main() {
	os.Exit(cli.Execute(os.Stdin, os.Stdout, os.Stderr).Value())
}
