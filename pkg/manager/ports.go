package manager

import (
	"fmt"
	"sync"
)

type portAllocator struct {
	mu       sync.Mutex
	allocated map[int]string // port -> instance name
	next      int
}

func newPortAllocator(min, max int) *portAllocator {
	return &portAllocator{
		allocated: make(map[int]string),
		next:      min,
	}
}

func (a *portAllocator) Allocate(name string, specificPort int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if specificPort != 0 {
		if a.allocated[specificPort] != "" {
			return 0, fmt.Errorf("port %d already allocated to %s", specificPort, a.allocated[specificPort])
		}
		a.allocated[specificPort] = name
		return specificPort, nil
	}

	// Find lowest free port
	for port := a.next; port <= 9000; port++ {
		if a.allocated[port] == "" {
			a.allocated[port] = name
			a.next = port + 1
			return port, nil
		}
	}

	// Wrap around
	for port := 8100; port < a.next; port++ {
		if a.allocated[port] == "" {
			a.allocated[port] = name
			a.next = port + 1
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range")
}

func (a *portAllocator) Free(port int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.allocated, port)
}

func (a *portAllocator) MarkAllocated(port int, name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.allocated[port] = name
}
