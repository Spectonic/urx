package urx

import "sync"

type liftedObservable struct {
	source privObservable
	op     Operator
}

type liftedSubscriber struct {
	source privSubscription
	op     Operator
	events chan Notification
	unsub chan interface{}
	cMutex sync.RWMutex
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
	for ev := range sub.source.Events() {
		sub.op.Notify(sub, ev)
	}
	close(sub.events)
}

func (sub *liftedSubscriber) Events() <-chan Notification {
	return sub.events
}

func (sub *liftedSubscriber) Unsubscribe() {
	sub.cMutex.RLock()
	close(sub.unsub)
	sub.cMutex.RUnlock()
	sub.cMutex.Lock()
	defer sub.cMutex.Unlock()
	if !sub.source.IsSubscribed() {
		return
	}
	sub.source.Unsubscribe()
	sub.callHooks()
}

func (sub *liftedSubscriber) IsSubscribed() bool {
	return sub.source.IsSubscribed() && !sub.finished
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
