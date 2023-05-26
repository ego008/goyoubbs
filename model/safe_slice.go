package model

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// ex:
// cs := NewConStrSlice()
//	cs.Append("foo")
//	cs.Append("bar")
//	cs.Append("qux")

type StrItems []string

func (s StrItems) Has(x string) bool {
	sl := len(s)
	for i := 0; i < sl; i++ {
		if s[i] == x {
			return true
		}
	}
	return false
}

func (s StrItems) Sort() {
	sort.Strings(s) // 升序
}

// ConStrSlice type that can be safely shared between goroutines.
type ConStrSlice struct {
	//sync.RWMutex
	mu    sync.Mutex
	items atomic.Value
	// items []string
}

// ConStrSliceItem contains the index/value pair of an item in a
// concurrent slice.
type ConStrSliceItem struct {
	Index int
	Value string
}

// NewConStrSlice creates a new concurrent slice.
func NewConStrSlice() *ConStrSlice {
	cs := &ConStrSlice{}
	cs.items.Store(make(StrItems, 0))

	return cs
}

// Append adds an item to the concurrent slice.
func (cs *ConStrSlice) Append(item string) {
	cs.mu.Lock()
	items := cs.items.Load().(StrItems)
	items = append(items, item)
	cs.items.Store(items)
	cs.mu.Unlock()
}

func (cs *ConStrSlice) Sort() {
	cs.mu.Lock()
	items := cs.items.Load().(StrItems)
	sort.Strings(items)
	cs.items.Store(items)
	cs.mu.Unlock()
}

// Copy other items to here
func (cs *ConStrSlice) Copy(items StrItems) {
	cs.mu.Lock()
	cs.items.Store(items)
	cs.mu.Unlock()
}

// Get an index
func (cs *ConStrSlice) Get(index int) string {
	items := cs.items.Load().(StrItems)
	if len(items) > index {
		return items[index]
	}
	return ""
}

// ItemInPrefix check this slice item is the prefix of given string [s]
func (cs *ConStrSlice) ItemInPrefix(s string) bool {
	//cs.RLock()
	//defer cs.RUnlock()
	items := cs.items.Load().(StrItems)
	il := len(items)
	for i := 0; i < il; i++ {
		if strings.HasPrefix(s, items[i]) {
			return true
		}
	}
	return false
}

// Contains check list Contains item
func (cs *ConStrSlice) Contains(s string) bool {
	items := cs.items.Load().(StrItems)
	il := len(items)
	for i := 0; i < il; i++ {
		if strings.Compare(s, items[i]) == 0 {
			return true
		}
	}
	return false
}

// ModGet an index
func (cs *ConStrSlice) ModGet(index int) string {
	//cs.RLock()
	//defer cs.RUnlock()
	items := cs.items.Load().(StrItems)
	return items[index%len(items)]
}

// Len is the number of items in the concurrent slice.
func (cs *ConStrSlice) Len() int {
	//cs.RLock()
	//defer cs.RUnlock()
	return len(cs.items.Load().(StrItems))
}

func (cs *ConStrSlice) KvEach(fn func(key int, value string)) int {
	items := cs.items.Load().(StrItems)
	il := len(items)
	for i := 0; i < il; i++ {
		fn(i, items[i])
	}
	return il
}

func (cs *ConStrSlice) Items() []string {
	return cs.items.Load().(StrItems)
}
