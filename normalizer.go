package main

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// URLSet is a set of *url.URLs, without repeats.
type URLSet struct {
	set map[string]*url.URL
}

// NewURLSet creates a new URLSet.
func NewURLSet() *URLSet {
	return &URLSet{
		set: make(map[string]*url.URL),
	}
}

// Add adds the given URLs into the set, hopefully eliminating repeats as it goes.
func (uset *URLSet) Add(urls ...*url.URL) {
	for _, u := range urls {
		uset.set[u.String()] = u
	}
}

// ToSlice returns all the elements of the set in the form of a slice
func (uset *URLSet) ToSlice() []*url.URL {
	uslice := make([]*url.URL, 0, len(uset.set))
	for _, u := range uset.set {
		uslice = append(uslice, u)
	}

	return uslice
}

// NormalizeURLs receives a slice of URLs in string form, detect whether they're
// from an individual ebook, author or a collection, applies the appropiate parser,
// and returns an *URLSet of the individual ebook files.
//
// The timer will be used to peace HTTP connections with the provided client.
// Before each connection, the timer will be waited for, and reset with the
// given duration after the body of the response has been read. The timer should
// have been properly initialized before calling this function, even if with an
// initial wait time of 0.
//
// All URLs returned are relative to the StandardEbooks main url.
func NormalizeURLs(rawURLs []string, formats string, connectionWait time.Duration, timer *time.Timer, client *http.Client) (*URLSet, error) {
	// Eliminate repeats in the raw URLs
	rawURLs = RemoveStringDuplicates(rawURLs)

	finalURLs := NewURLSet()

	ebookParser, err := NewEbookPageParser(formats)
	if err != nil {
		return finalURLs, fmt.Errorf("while creating EbookPageParser: %v", err)
	}
	collectionParser := NewCollectionPageParser()
	authorParser := NewAuthorPageParser()

	for _, rawURL := range rawURLs {
		// Check if the URL is from StandardEbooks at all
		if !StandardEbooksMainRegex.MatchString(rawURL) {
			return finalURLs, fmt.Errorf("%s is not a valid StandardEbook book", rawURL)
		}

		if EbookURLRegex.MatchString(rawURL) { // A single ebook
			err = func() error {
				<-timer.C

				resp, err := client.Get(rawURL)
				if err != nil {
					return fmt.Errorf("while getting %s: %v", rawURL, err)
				}
				defer resp.Body.Close()

				urls, err := ebookParser.Parse(resp.Body)
				if err != nil {
					return fmt.Errorf("while parsing %s: %v", rawURL, err)
				}

				finalURLs.Add(urls...)

				timer.Reset(connectionWait)

				return nil
			}()
			if err != nil {
				return finalURLs, err
			}

		} else if CollectionURLRegex.MatchString(rawURL) { // A collection of ebooks
			err = func() error {
				// First getting the individual books
				<-timer.C

				resp, err := client.Get(rawURL)
				if err != nil {
					return fmt.Errorf("while getting %s: %v", rawURL, err)
				}
				defer resp.Body.Close()

				booksURLs, err := collectionParser.Parse(resp.Body)
				if err != nil {
					return fmt.Errorf("while parsing %s: %v", rawURL, err)
				}

				timer.Reset(connectionWait)

				// For each book page, get its files
				for _, bookURL := range booksURLs {
					err = func(bookURL *url.URL) error {
						completeBookURL := StandardEbooksMainURL.ResolveReference(bookURL)

						<-timer.C

						resp, err := client.Get(completeBookURL.String())
						if err != nil {
							return fmt.Errorf("while getting %s (collection: %s): %v", bookURL, rawURL, err)
						}
						defer resp.Body.Close()

						urls, err := ebookParser.Parse(resp.Body)
						if err != nil {
							return fmt.Errorf("while parsing %s (collection: %s): %v", bookURL, rawURL, err)
						}
						timer.Reset(connectionWait)

						finalURLs.Add(urls...)

						return nil
					}(bookURL)
					if err != nil {
						break
					}
				}
				return err
			}()
			if err != nil {
				return finalURLs, err
			}

		} else if AuthorURLRegex.MatchString(rawURL) { // An author page
			err = func() error {
				// First getting the individual books
				<-timer.C
				resp, err := client.Get(rawURL)
				if err != nil {
					return fmt.Errorf("while getting %s: %v", rawURL, err)
				}
				defer resp.Body.Close()

				booksURLs, err := authorParser.Parse(resp.Body)
				if err != nil {
					return fmt.Errorf("while parsing %s: %v", rawURL, err)
				}

				timer.Reset(connectionWait)

				// For each book page, get its files
				for _, bookURL := range booksURLs {
					err = func(bookURL *url.URL) error {
						completeBookURL := StandardEbooksMainURL.ResolveReference(bookURL)

						<-timer.C
						resp, err := client.Get(completeBookURL.String())
						if err != nil {
							return fmt.Errorf("while getting %s (author: %s): %v", bookURL, rawURL, err)
						}
						defer resp.Body.Close()

						urls, err := ebookParser.Parse(resp.Body)
						if err != nil {
							return fmt.Errorf("while parsing %s (author: %s): %v", bookURL, rawURL, err)
						}

						timer.Reset(connectionWait)

						finalURLs.Add(urls...)

						return nil
					}(bookURL)
					if err != nil {
						break
					}
				}

				return err
			}()
			if err != nil {
				return finalURLs, err
			}
		} else { // Default: not a valid URL
			return finalURLs, fmt.Errorf("%s was not recognized as a valid URL format", rawURL)
		}
	}

	return finalURLs, nil
}
