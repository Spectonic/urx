package urx

import "sync"

type liftedObservable struct {
	source privObservable
	op     Operator
}

type liftedSubscriber struct {
	source   privSubscription
	op       Operator
	events   chan Notification
	unsub    chan interface{}
	cMutex   sync.RWMutex
	uMutex   sync.Mutex
	unsubbed bool
	hooks
}

func (lifted *liftedObservable) privSubscribe() (sub privSubscription) {
	out := &liftedSubscriber{source: lifted.source.privSubscribe(), op: lifted.op, events: make(chan Notification), unsub: make(chan interface{})}
	go out.pump()
	sub = out
	return
}

func (lifted *liftedObservable) Lift(op Operator) (obs privObservable) {
	obs = &liftedObservable{source: lifted, op: op}
	return
}

func (sub *liftedSubscriber) pump() {
	defer close(sub.events)
	for ev := range sub.source.Events() {
		sub.op.Notify(sub, ev)
	}
}

func (sub *liftedSubscriber) Events() <-chan Notification {
	return sub.events
}

func (sub *liftedSubscriber) Unsubscribe() {
	sub.uMutex.Lock()
	if sub.unsubbed {
		sub.uMutex.Unlock()
		return
	}
	close(sub.unsub)
	sub.unsubbed = true
	sub.source.Unsubscribe()
	sub.uMutex.Unlock()
	sub.cMutex.Lock()
	defer sub.cMutex.Unlock()
	sub.callHooks()
}

func (sub *liftedSubscriber) IsSubscribed() bool {
	return !sub.unsubbed && sub.source.IsSubscribed() && !sub.finished
}

func (sub *liftedSubscriber) Notify(not Notification) {
	sub.cMutex.RLock()
	select {
	case sub.events <- not:
		sub.cMutex.RUnlock()
		if not.Type == OnComplete {
			sub.Unsubscribe()
		}
	case <-sub.unsub:
		sub.cMutex.RUnlock()
		return
	}
}
