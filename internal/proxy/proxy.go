// Package proxy provides a http proxy
package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/go-chi/chi"
	"go.uber.org/multierr"
)

// Proxy enables a http proxy
type Proxy struct {
	r        *chi.Mux
	srv      *http.Server
	wg       *sync.WaitGroup
	backends map[string]*BackendHost
}

// BackendHost is a host we want to proxy to
type BackendHost struct {
	// address of the backend host
	Addr string
	// if true we use http connection
	Insecure bool
}

// New returns an initialized, but unstarted proxy
func New(addr string, backends map[string]*BackendHost) *Proxy {
	proxy := &Proxy{
		r:        chi.NewRouter(),
		wg:       &sync.WaitGroup{},
		backends: backends,
	}
	proxy.r.HandleFunc("/*", proxy.handle)
	proxy.srv = &http.Server{Addr: addr, Handler: proxy.r}
	return proxy
}

// Run starts the http proxy
func (p *Proxy) Run(ctx context.Context) error {
	var (
		err    = make(chan error, 1)
		errors []error
	)
	p.wg.Add(1)
	defer p.wg.Wait()
	go func() {
		defer p.wg.Done()
		err <- p.srv.ListenAndServe()
	}()
	<-ctx.Done()
	closeErr := p.srv.Close()
	errors = append(errors, closeErr)
	errors = append(errors, <-err)
	return multierr.Combine(errors...)
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed to parse hostname"))
		return
	}
	backend := p.backends[host]
	if backend == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("no backend matching hostname"))
		return
	}
	callURL := func() string {
		var prefix string
		if backend.Insecure {
			prefix = "http://"
		} else {
			prefix = "https://"
		}
		return prefix + backend.Addr + r.RequestURI
	}()
	target, err := url.Parse(callURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to parse callURL"))
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

func getHostName(host string) string {
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		if len(parts) == 0 {
			return ""
		}
		return parts[0]
	}
	return host
}
