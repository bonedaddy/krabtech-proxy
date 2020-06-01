package proxy

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

// DefaultOptions returns a generic, default options
// mainly useful for testing
func DefaultOptions(withBackend bool) *Options {
	var backend map[string]*BackendHost
	if withBackend {
		backend = map[string]*BackendHost{
			"foobar": {
				Addr:     "localhost:6666",
				Insecure: true,
			},
		}
	}
	return &Options{
		ListenAddress:    ":6665",
		LogFile:          "proxy.log",
		Backends:         backend,
		BasicAuthEnabled: true,
		BasicAuthRealm:   "krabtech-proxy",
		BasicAuthUsers:   map[string]string{"user": "pass"},
	}
}
