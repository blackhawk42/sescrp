package main

import (
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

// TesterFunction is a function that takes a string an preforms a test on it,
// returning a boolean
type TesterFunction func(string) bool

// FormatsTestersMap is a map that maps a string key representing a format with a tester function.
type FormatsTestersMap map[string]TesterFunction

// GetKeys gets the sorted keys of the FormatsTestersMap
func (fm FormatsTestersMap) GetKeys() []string {
	keys := make([]string, 0, len(fm))

	for k := range fm {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// FormatsTesters maps a format name to a function that tastes if a string conforms
// to that format, according to naming conventions of files in Standard Ebooks.
var FormatsTesters FormatsTestersMap = map[string]TesterFunction{
	"epub": func(name string) bool {
		return strings.HasSuffix(name, ".epub") && !strings.HasSuffix(name, ".kepub.epub") && !strings.HasSuffix(name, "_advanced.epub")
	},
	"azw3": func(name string) bool {
		return strings.HasSuffix(name, ".azw3")
	},
	"kepub": func(name string) bool {
		return strings.HasSuffix(name, ".kepub.epub")
	},
	"aepub": func(name string) bool {
		return strings.HasSuffix(name, "_advanced.epub")
	},
}

// EbookPageParser parses the page of an individual ebook.
type EbookPageParser struct {
	extensionsTesters []TesterFunction
}

// NewEbookPageParser creates a new EbookPageParser.
//
// extensions should be a comma-separated list with any of the supported formats,
// e. g., "epub,kepub,azw3". An error will be returned if an unsupported format is
// passed.
func NewEbookPageParser(extensions string) (*EbookPageParser, error) {
	extensionsSlice := strings.Split(extensions, ",")
	extensionsTesters := make([]TesterFunction, 0, len(extensionsSlice))

	for _, ext := range extensionsSlice {
		fun, ok := FormatsTesters[ext]
		if !ok {
			return nil, fmt.Errorf("the extension \"%s\" is not supported", ext)
		}

		extensionsTesters = append(extensionsTesters, fun)
	}

	return &EbookPageParser{
		extensionsTesters: extensionsTesters,
	}, nil
}

// Parse parses a given ebook page, provided through an io.Reader.
//
// It returns a slice of successfully parsed *url.URLs and an error, if any. No
// new HTTP connections are made.
//
// All URLs returned are relative to the StandardEbooks main url.
func (ebookParser *EbookPageParser) Parse(htmlReader io.Reader) ([]*url.URL, error) {
	doc, err := html.Parse(htmlReader)
	if err != nil {
		return nil, err
	}

	finalUrls := make([]*url.URL, 0, len(ebookParser.extensionsTesters))
	err = nil

	var parseF func(*html.Node)
	parseF = func(n *html.Node) {
		// Detect links
		if n.Type == html.ElementNode && n.Data == "a" {
			// Iterate attributes in search of an href
			for _, attr := range n.Attr {
				// Add url if it matches one of the active formats
				if attr.Key == "href" && ebookParser.urlMatches(attr.Val) {
					newURL, localError := url.Parse(attr.Val)
					if localError != nil {
						err = fmt.Errorf("while processing %s: %v", attr.Val, localError)
						return
					}

					finalUrls = append(finalUrls, newURL)
				}
			}
		}

		// Recursive calls to do a depth-first search
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseF(c)
		}
	}

	// Start the search
	parseF(doc)

	return finalUrls, err
}

// Check if the given URL (in string form) matches any of the active extensions.
func (ebookParser *EbookPageParser) urlMatches(url string) bool {
	for _, test := range ebookParser.extensionsTesters {
		if test(url) {
			return true
		}
	}

	return false
}

// CollectionPageParser parses the page of an entire collection
type CollectionPageParser struct {
}

// NewCollectionPageParser creates a new CollectionPageParser
func NewCollectionPageParser() *CollectionPageParser {
	return new(CollectionPageParser)
}

// Parse parses a given collection page, provided through an io.Reader.
//
// It returns a slice with the *url.URLs of all individual book pages. No HTTP
// connection is actually made.
//
// All URLs returned are relative to the StandardEbooks main url.
func (collectionParser *CollectionPageParser) Parse(htmlReader io.Reader) ([]*url.URL, error) {
	doc, err := html.Parse(htmlReader)
	if err != nil {
		return nil, err
	}

	finalUrls := make([]*url.URL, 0)
	err = nil

	var parseF func(n *html.Node)
	parseF = func(n *html.Node) {
		// Detect links
		if n.Type == html.ElementNode && n.Data == "a" {
			// This link must be inside a <p> with no attributes, which is inside a <li>
			if n.Parent.Type == html.ElementNode && n.Parent.Data == "p" && len(n.Parent.Attr) == 0 && n.Parent.Parent.Type == html.ElementNode && n.Parent.Parent.Data == "li" {
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						newURL, localErr := url.Parse(attr.Val)
						if localErr != nil {
							err = fmt.Errorf("while processing %s: %v", attr.Val, localErr)
							return
						}

						finalUrls = append(finalUrls, newURL)
					}
				}
			}
		}

		// Recursive calls to do a depth-first search
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseF(c)
		}
	}

	parseF(doc)

	return finalUrls, err
}

// AuthorPageParser parses the page of an author.
type AuthorPageParser struct {
}

// NewAuthorPageParser creates a new AuthorPageParser
func NewAuthorPageParser() *AuthorPageParser {
	return new(AuthorPageParser)
}

// Parse parses a given author page, provided through an io.Reader.
//
// It returns a slice with the *url.URLs of all individual book pages. No HTTP
// connection is actually made.
//
// All URLs returned are relative to the StandardEbooks main url.
func (authorParser *AuthorPageParser) Parse(htmlReader io.Reader) ([]*url.URL, error) {
	doc, err := html.Parse(htmlReader)
	if err != nil {
		return nil, err
	}

	finalUrls := make([]*url.URL, 0)
	err = nil

	var parseF func(n *html.Node)
	parseF = func(n *html.Node) {
		// Detect links
		if n.Type == html.ElementNode && n.Data == "a" {
			// This link must be inside a <p> with no attributes, which is inside a <li>.
			// As of right now, this seems to be the same rule as for collections,
			// but it's implemented on its own, in case this canges in the future.
			if n.Parent.Type == html.ElementNode && n.Parent.Data == "p" && len(n.Parent.Attr) == 0 && n.Parent.Parent.Type == html.ElementNode && n.Parent.Parent.Data == "li" {
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						newURL, localErr := url.Parse(attr.Val)
						if localErr != nil {
							err = fmt.Errorf("while processing %s: %v", attr.Val, localErr)
							return
						}

						finalUrls = append(finalUrls, newURL)
					}
				}
			}
		}

		// Recursive calls to do a depth-first search
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseF(c)
		}
	}

	parseF(doc)

	return finalUrls, err
}
