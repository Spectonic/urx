package urx

import "sync"

type hooks struct {
	slice    []CompleteHook
	m        sync.Mutex
	finished bool
}

func (h *hooks) Add(hook CompleteHook) {
	h.m.Lock()
	defer h.m.Unlock()

	if h.finished {
		return
	}

	h.slice = append(h.slice, hook)
	return
}

func (h *hooks) callHooks() {
	h.m.Lock()
	defer h.m.Unlock()

	for i := range h.slice {
		h.slice[i]()
	}
	h.slice = nil
	h.finished = true
}
