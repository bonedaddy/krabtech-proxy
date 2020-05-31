package proxy

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi"
)

func TestProxy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testServer := newTestServer(t, ":6666")
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	wg.Add(1)
	go func() { testServer.run(ctx); wg.Done() }()

	testProxy := New(":6665", map[string]*BackendHost{
		"foobar": &BackendHost{
			Insecure: true,
			Addr:     "localhost:6666",
		},
	})
	wg.Add(1)
	go func() { testProxy.Run(ctx); wg.Done() }()
	req, err := http.NewRequest("POST", "http://localhost:6666/random", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "foobar"
	client := http.Client{}
	if _, err := client.Do(req); err != nil {
		t.Fatal(err)
	}
	if testServer.count.count == 0 {
		t.Fatal("bad count")
	}
	time.Sleep(time.Second * 2)
	cancel()
}

type countWrite struct {
	count int
	mux   sync.RWMutex
}

type testServer struct {
	srv   *http.Server
	count *countWrite
}

func newTestServer(t *testing.T, addr string) *testServer {
	srv := &testServer{
		count: &countWrite{},
	}
	r := chi.NewRouter()
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		srv.count.mux.Lock()
		defer srv.count.mux.Unlock()
		srv.count.count++
	})
	srv.srv = &http.Server{Addr: addr, Handler: r}
	return srv
}

func (ts *testServer) run(ctx context.Context) {
	go func() {
		ts.srv.ListenAndServe()
	}()
	<-ctx.Done()
	ts.srv.Close()
}
