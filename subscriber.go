package urx

import (
	"sync"
	"time"
)

type Subscription interface {
	Events() <- chan Notification
	Unsubscribe()
	Values() <- chan interface{}
	Error() <- chan error
	Complete() <- chan interface{}
	RootSubscriber
}

type wrappedSubscription struct {
	sub privSubscription
	source chan Notification
	values, complete chan interface{}
	error chan error
	pumping bool
	mutex sync.RWMutex
}

func (s wrappedSubscription) init() *wrappedSubscription {
	s.error = make(chan error)
	s.values = make(chan interface{})
	s.complete = make(chan interface{})
	s.source = make(chan Notification)
	return &s
}

func (s *wrappedSubscription) pump() {
pump_loop:
	for e := range s.sub.Events() {
		switch e.Type {
		case OnNext:
			select {
			case s.source <- e:
			case s.values <- e.Body:
			}
		case OnError:
			select {
			case s.source <- e:
			case s.error <- e.Body.(error):
			}
		case OnComplete:
			break pump_loop
		}
	}
}

func (s *wrappedSubscription) Events() <- chan Notification {
	s.initIfNeeded(true)
	return s.source
}

func (s *wrappedSubscription) Unsubscribe() {
	s.sub.Unsubscribe()
}

func (s *wrappedSubscription) initIfNeeded(allEvents bool) {
	s.mutex.RLock()
	if !s.pumping {
		s.mutex.RUnlock()
		s.mutex.Lock()
		s.pumping = true
		s.mutex.Unlock()
		go func() {
			if allEvents {
				s.source <- Start()
			}
			s.pump()
			select {
			case s.source <- Complete():
			case s.complete <- nil:
			case <-time.After(time.Millisecond):
			}
			close(s.error)
			close(s.values)
			close(s.complete)
			close(s.source)
		}()
	} else {
		s.mutex.RUnlock()
	}
}

func (s *wrappedSubscription) Values() <- chan interface{} {
	s.initIfNeeded(false)
	return s.values
}

func (s *wrappedSubscription) Error() <- chan error {
	s.initIfNeeded(false)
	return s.error
}

func (s *wrappedSubscription) Complete() <- chan interface{} {
	s.initIfNeeded(false)
	return s.complete
}

func (s *wrappedSubscription) IsSubscribed() bool {
	return s.sub.IsSubscribed()
}

func (s *wrappedSubscription) Add(hook CompleteHook) {
	s.sub.Add(hook)
}
