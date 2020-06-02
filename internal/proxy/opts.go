package proxy

import (
	"crypto/tls"
)

var (
	tlsConfig = tls.Config{
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		//PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			// http/2 mandated supported cipher
			// unforunately this is a less secure cipher
			// but specifying it first is the only way to accept
			// http/2 connections without go throwing an error
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			// super duper secure ciphers
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		MinVersion: tls.VersionTLS11,
	}
)

// Options used in configuring the proxy
type Options struct {
	ListenAddress    string
	LogFile          string
	Backends         map[string]*BackendHost
	BasicAuthEnabled bool
	BasicAuthRealm   string
	BasicAuthUsers   map[string]string
}

// BackendHost is a host we want to proxy to
type BackendHost struct {
	// address of the backend host
	Addr string
	// if true we use http connection
	Insecure bool
}

// TLSOpts allows configuring the tls endpoint for the proxy
type TLSOpts struct {
	cfg      *tls.Config
	CertFile string
	KeyFile  string
}

// DefaultOptions returns a generic, default options
// mainly useful for testing
func DefaultOptions() *Options {
	return &Options{
		ListenAddress:    ":6665",
		LogFile:          "proxy.log",
		BasicAuthEnabled: true,
		BasicAuthRealm:   "krabtech-proxy",
		BasicAuthUsers:   map[string]string{"user": "pass"},
	}
}
