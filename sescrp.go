package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// StandardEbooksMainURL is the main url for the StandardEbooks website, for use
// in things like URL parsing.
var StandardEbooksMainURL = MustParseURL("https://standardebooks.org")

// Regular expressions used for things like URL validation and selection of appropiate
// parsers.
var (
	StandardEbooksMainRegex = regexp.MustCompile(`https://standardebooks.org/.*[/]?$`)
	EbookURLRegex           = regexp.MustCompile(`https://standardebooks.org/ebooks/[A-Za-z\-]+/.*[/]?$`)
	AuthorURLRegex          = regexp.MustCompile(`https://standardebooks.org/ebooks/[A-Za-z\-]+[/]?$`)
	CollectionURLRegex      = regexp.MustCompile(`https://standardebooks.org/collections/.*[/]?$`)
)

// Flag defaults
var (
	DefaultExtensions     []string = []string{"epub", "azw3", "kepub", "epub3"}
	DefaultBasedir        string   = "."
	DefaultConnectionWait int64    = 1
	DefaultTrimKepub      bool     = false
)

// Flag variables
var (
	extensions     = flag.String("extensions", strings.Join(DefaultExtensions, ","), "`extensions` to look for in files, separated by commas; by default, and as of this writing, all StandardEbook formats should be supported")
	basedir        = flag.String("dir", DefaultBasedir, "base `directory` where to download the files, and create it if necessary; a \".\" means the current directory")
	connectionWait = flag.Int64("connection-wait", DefaultConnectionWait, "how many `seconds` to wait between *every* required HTTP connection, including parsing (*not* just between individual ebook file downloads); can be set to 0, but let's try to be nice to StandardEbook servers, if possible")
	trimKepub      = flag.Bool("trim-kepub", DefaultTrimKepub, "download kepub files with the extension \".kepub\", instead of \".kepub.epub\"")
)

func main() {
	// Flag and initial setup

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [FLAGS] URL [URL...]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(flag.CommandLine.Output(), "Scrap ebook files from StandardEbooks.\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "As of this date, StandardEbooks robots.txt is intentionally left blank (ha!), which is great on their part. Nevertheless, in consideration of not being an abusive scrapper, an effort was made to keep all connections one at a time and with a timer between them.\n\n")

		flag.PrintDefaults()
	}

	flag.Parse()

	// No arguments are equivalent to invoking help
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	if *connectionWait < 0 {
		fmt.Fprintf(os.Stderr, "error: time between connections can't be a negative number\n")
		flag.Usage()
		os.Exit(2)
	}
	duration := time.Duration(*connectionWait) * time.Second

	if *basedir == "" {
		fmt.Fprintf(os.Stderr, "error: base directory can't be empty\n")
		flag.Usage()
		os.Exit(2)
	}

	var err error
	*basedir, err = filepath.Abs(*basedir)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll(*basedir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	// Client to use in the connections
	client := &http.Client{}

	// Timer initially set to expire inmediately
	timer := time.NewTimer(0)
	urls, err := NormalizeURLs(flag.Args(), *extensions, duration, timer, client)
	if err != nil {
		log.Fatal(err)
	}

	for _, ebookURL := range urls.ToSlice() {
		func(ebookURL *url.URL) {
			ebookURL = StandardEbooksMainURL.ResolveReference(ebookURL)

			filename := path.Base(ebookURL.String())

			if *trimKepub && strings.HasSuffix(filename, ".kepub.epub") {
				filename = strings.TrimSuffix(filename, ".epub")
			}

			absFilename := filepath.Join(*basedir, filename)

			f, err := os.Create(absFilename)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			<-timer.C

			log.Printf("downloading %s to %s", ebookURL, absFilename)
			resp, err := client.Get(ebookURL.String())
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			io.Copy(f, resp.Body)

			timer.Reset(duration)
		}(ebookURL)
	}

}
