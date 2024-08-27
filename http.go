package gira

import "net/http"

type HttpHandler interface {
	HttpHandler() http.Handler
}
