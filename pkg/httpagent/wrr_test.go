package httpagent

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"sync"
	"testing"
)

func TestWRRSanity(t *testing.T) {
	wrr := NewWeightRR()
	wrr.AddItem("A", 1)
	wrr.AddItem("B", 1)
	wrr.AddItem("C", 2)

	wg := &sync.WaitGroup{}
	ch := make(chan interface{}, 1)
	quit := make(chan struct{})
	count := make(map[string]int)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			ch <- wrr.GetItem()

			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(quit)
	}()

	for done := false; !done; {
		select {
		case <-quit:
			done = true
		case v := <-ch:
			count[v.(string)] += 1
		}
	}

	assert.Equal(t, 25, count["A"])
	assert.Equal(t, 25, count["B"])
	assert.Equal(t, 50, count["C"])
}

func TestWRRSmoth(t *testing.T) {
	wrr := NewWeightRR()
	wrr.AddItem("A", 1)
	wrr.AddItem("B", 1)
	wrr.AddItem("C", 2)

	var output string
	for i := 0; i < 40; i++ {
		s := wrr.GetItem().(string)
		output = output + s
	}

	expected := strings.Repeat("CABC", 10)
	assert.Equal(t, expected, output)
}
