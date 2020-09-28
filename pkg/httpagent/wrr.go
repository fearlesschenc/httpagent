package httpagent

import (
	"sync"
)

type WeightRRItem struct {
	item   interface{}
	weight int
}

type WeightRR struct {
	mutex    *sync.Mutex
	items    []WeightRRItem
	cw       int
	selected int
	maxW     int
	gcdW     int
}

func NewWeightRR() *WeightRR {
	return &WeightRR{
		mutex:    &sync.Mutex{},
		cw:       0,
		selected: -1,
		maxW:     0,
		gcdW:     0,
	}
}

func gcd(x, y int) int {
	var t int
	for {
		t = x % y
		if t > 0 {
			x = y
			y = t
		} else {
			return y
		}
	}
}

func (w *WeightRR) AddItem(item interface{}, weight int) bool {
	if weight < 0 {
		return false
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.items = append(w.items, WeightRRItem{
		item:   item,
		weight: weight,
	})

	if weight > w.maxW {
		w.maxW = weight
	}

	w.gcdW = gcd(w.gcdW, weight)
	w.cw = 0
	w.selected = -1

	return true
}

func (w *WeightRR) GetItem() interface{} {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for {
		w.selected = (w.selected + 1) % len(w.items)
		if w.selected == 0 {
			w.cw = w.cw - w.gcdW
			if w.cw <= 0 {
				w.cw = w.maxW
				if w.cw == 0 {
					return nil
				}
			}
		}

		if w.items[w.selected].weight >= w.cw {
			return w.items[w.selected].item
		}
	}
}
