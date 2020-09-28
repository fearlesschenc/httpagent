package httpagent

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func createServer(port int, parent context.Context) {
	ctx, _ := context.WithCancel(parent)

	// mock a server
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "OK")
		})

		s := &http.Server{
			Addr:    fmt.Sprintf("127.0.0.1:%d", port),
			Handler: mux,
		}

		go func() {
			s.ListenAndServe()
		}()

		<-ctx.Done()
		s.Shutdown(context.Background())
	}()
}

func TestAgent(t *testing.T) {
	agent := New(&http.Client{
		Timeout: 1 * time.Second,
	}, &Config{
		MaxTry:      3,
		MaxFails:    3,
		FailTimeout: 1 * time.Second,
	})
	agent.AddServer("127.0.0.1", 12345, 1)
	agent.AddServer("127.0.0.1", 12346, 1)

	// create server
	ctx, cancelFunc := context.WithCancel(context.Background())
	createServer(12345, ctx)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Error("create request error")
	}
	for i := 0; i < 5; i++ {
		_, err := agent.Do(req)
		if err != nil {
			t.Error("request error")
		}
	}
	assert.Equal(t, false, agent.stat[0].down)
	assert.Equal(t, true, agent.stat[1].down)

	// let fail timeout
	time.Sleep(1 * time.Second)

	// still not working
	assert.Equal(t, true, agent.stat[1].down)

	// start server on 12346
	createServer(12346, ctx)
	// test again
	for i := 0; i < 5; i++ {
		_, err := agent.Do(req)
		if err != nil {
			t.Error("request error")
		}
	}
	assert.Equal(t, false, agent.stat[0].down)
	// should be working now
	assert.Equal(t, false, agent.stat[1].down)

	// close servers
	cancelFunc()
}
