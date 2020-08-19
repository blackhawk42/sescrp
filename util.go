package main

import (
	"net/url"
)

// IsIn checks if a given string exists in a slice of strings
func IsIn(value string, slice []string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}

	return false
}

// MustParseURL attempts to parse an *url.URL from a string, with panic on error.
func MustParseURL(rawURL string) *url.URL {
	url, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return url
}

// RemoveStringDuplicates remove duplicated string elements from a slice of strings
func RemoveStringDuplicates(slice []string) []string {
	returnSlice := make([]string, 0)
	seen := make(map[string]struct{})

	for _, s := range slice {
		if _, wasThere := seen[s]; !wasThere {
			returnSlice = append(returnSlice, s)
			seen[s] = struct{}{}
		}
	}

	return returnSlice
}
