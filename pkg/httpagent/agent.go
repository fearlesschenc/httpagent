package httpagent

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Server struct {
	host  string
	port  int
	index int
}

type ServerStat struct {
	down    bool
	fails   int
	tryTime time.Time
}

type Config struct {
	MaxTry      int
	MaxFails    int
	FailTimeout time.Duration
}

type Agent struct {
	c *Config

	mutex  *sync.Mutex
	wrr    *WeightRR
	client *http.Client
	stat   []*ServerStat
}

func New(client *http.Client, c *Config) *Agent {
	mutex := &sync.Mutex{}

	return &Agent{
		mutex:  mutex,
		wrr:    NewWeightRR(),
		c:      c,
		client: client,
	}
}

func (agent *Agent) AddServer(host string, port int, weight int) bool {
	// add host/port in wrr
	ok := agent.wrr.AddItem(&Server{host: host, port: port, index: len(agent.stat)}, weight)
	if !ok {
		return false
	}

	agent.mutex.Lock()
	defer agent.mutex.Unlock()

	// add server stat
	agent.stat = append(agent.stat, &ServerStat{
		fails:   0,
		down:    false,
		tryTime: time.Now(),
	})

	return true
}

func (agent *Agent) markFail(index int) {
	agent.mutex.Lock()
	defer agent.mutex.Unlock()

	// add fail
	agent.stat[index].fails += 1
	// mark down
	if agent.stat[index].fails >= agent.c.MaxFails {
		agent.stat[index].down = true
		agent.stat[index].tryTime = time.Now().Add(agent.c.FailTimeout)
	}
}

func (agent *Agent) markOK(index int) {
	agent.mutex.Lock()
	defer agent.mutex.Unlock()

	// mark server down
	agent.stat[index].fails = 0
	agent.stat[index].down = false
}

func (agent *Agent) doRequest(host string, port int, req *http.Request) (*http.Response, error) {
	u, err := url.Parse(fmt.Sprintf("http://%s:%d%s", host, port, req.URL.Path))
	if err != nil {
		return nil, err
	}

	req.URL = u
	return agent.client.Do(req)
}

func (agent *Agent) Do(req *http.Request) (*http.Response, error) {
	for try := 0; try <= agent.c.MaxTry; {
		server := agent.wrr.GetItem().(*Server)

		if agent.stat[server.index].down {
			if agent.stat[server.index].tryTime.After(time.Now()) {
				continue
			}

			// try
			res, err := agent.doRequest(server.host, server.port, req)
			// refresh tryTime
			if err != nil {
				agent.markFail(server.index)
				continue
			}

			// rdy to go
			agent.markOK(server.index)
			return res, nil
		}

		res, err := agent.doRequest(server.host, server.port, req)
		if err != nil {
			// mark server fail
			agent.markFail(server.index)

			if try < agent.c.MaxTry-1 {
				try += 1
				continue
			}

			return nil, err
		}

		return res, nil
	}

	// impossible
	return nil, errors.New("No valid host")
}
