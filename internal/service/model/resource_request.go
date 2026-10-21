package model

import "net/url"

type ResourceURIRequest struct {
	URI    string
	Parsed *url.URL
}
