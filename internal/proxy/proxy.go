// Package proxy provides a http proxy
package proxy

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/oxtoacart/bpool"

	"github.com/go-chi/chi/middleware"
	"go.bobheadxi.dev/zapx/zapx"

	"github.com/go-chi/chi"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// Proxy enables a http proxy
type Proxy struct {
	r        *chi.Mux
	srv      *http.Server
	wg       *sync.WaitGroup
	backends map[string]*BackendHost
	logger   *zap.Logger
}

// New returns an initialized, but unstarted proxy
func New(opts *Options) *Proxy {
	logger, err := zapx.New(opts.LogFile, false)
	if err != nil {
		panic(err)
	}
	proxy := &Proxy{
		r:        chi.NewRouter(),
		wg:       &sync.WaitGroup{},
		backends: opts.Backends,
		logger:   logger.Named("proxy"),
	}
	if opts.BasicAuthEnabled {
		proxy.r.Use(middleware.BasicAuth(opts.BasicAuthRealm, opts.BasicAuthUsers))
	}
	proxy.r.Use(
		middleware.RequestID,
		middleware.RealIP,
		NewMiddleware(proxy.logger.Named("http.middleware")),
		middleware.Recoverer,
	)
	proxy.r.HandleFunc("/*", proxy.handle)
	proxy.srv = &http.Server{Addr: opts.ListenAddress, Handler: proxy.r}
	return proxy
}

// Run starts the http proxy
func (p *Proxy) Run(ctx context.Context, tlsOpts *TLSOpts) error {
	var (
		err     = make(chan error, 1)
		errors  []error
		runFunc = func() error {
			if tlsOpts != nil {
				p.srv.TLSConfig = tlsOpts.cfg
				return p.srv.ListenAndServeTLS(tlsOpts.CertFile, tlsOpts.KeyFile)
			}
			return p.srv.ListenAndServe()
		}
	)
	p.wg.Add(1)
	defer p.wg.Wait()
	go func() {
		defer p.wg.Done()
		err <- runFunc()
	}()
	<-ctx.Done()
	closeErr := p.srv.Close()
	errors = append(errors, closeErr)
	errors = append(errors, <-err)
	return multierr.Combine(errors...)
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	if p.backends == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("no backends configured"))
		return
	}
	host := r.Host
	if host == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("failed to parse hostname"))
		return
	}
	backend := p.backends[getHostName(host)]
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
	proxy := p.newProxy(target, r)
	proxy.ServeHTTP(w, r)
}

func (p *Proxy) newProxy(target *url.URL, r *http.Request) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Path = target.Path
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
		},
		ErrorLog: func() *log.Logger {
			slog, err := zap.NewStdLogAt(p.logger, zap.ErrorLevel)
			if err != nil {
				return log.New(os.Stdout, "", log.LstdFlags)
			}
			return slog
		}(),
		BufferPool: bpool.NewBytePool(10*1024, 10*1024),
	}
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
