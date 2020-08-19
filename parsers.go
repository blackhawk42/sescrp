package main

import (
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// EbookPageParser parses the page of an individual ebook.
type EbookPageParser struct {
	extensions []string
}

// NewEbookPageParser creates a new EbookPageParser.
//
// extensions should be a comma-separated list with any of the supported formats,
// e. g., "epub,kepub,azw3". An error will be returned if an unsupported format is
// passed.
func NewEbookPageParser(extensions string) (*EbookPageParser, error) {
	extensionsSlice := strings.Split(extensions, ",")

	for i, ext := range extensionsSlice {
		switch ext {
		case "epub":
			extensionsSlice[i] = ".epub"
		case "azw3":
			extensionsSlice[i] = ".azw3"
		case "kepub":
			extensionsSlice[i] = ".kepub.epub"
		case "epub3":
			extensionsSlice[i] = ".epub3"
		default:
			return nil, fmt.Errorf("the extension \"%s\" is not supported", ext)
		}
	}

	return &EbookPageParser{
		extensions: extensionsSlice,
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

	finalUrls := make([]*url.URL, 0, len(ebookParser.extensions))
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
	exts := strings.Split(path.Base(url), ".")

	if len(exts) < 2 { // no extensions
		return false
	} else if len(exts) > 2 { // multiple extensions
		return IsIn("."+strings.Join(exts[1:], "."), ebookParser.extensions)
	} else {
		return IsIn("."+exts[1], ebookParser.extensions)
	}
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
