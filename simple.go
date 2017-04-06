package urx

import (
	"sync"
)

// The simple observable is simply a function which takes a subscriber and provides it with data
type simpleObservable struct {
	onSub *func(Subscriber)
}

// this creates a subscription (by calling the simpleObservable function immediately)
func (obs simpleObservable) privSubscribe() privSubscription {
	//first, create a subscriber/observer combo
	outSub := initSimpleSubscriber()
	outSub.mangleError = true
	go outSub.Notify(Start())
	f := *obs.onSub
	go f(outSub)
	return outSub
}

// applies an operator to the observable such that subscriptions to the resulting observable flow through the operator
func (obs simpleObservable) Lift(op Operator) (newObs privObservable) {
	newObs = &liftedObservable{obs, op}
	return
}

type simpleSubscriber struct {
	//notifications from source are written here
	out chan Notification
	unsub chan interface{}
	//used to write up to a parent when an unsubscription
	hooks
	lock         sync.RWMutex
	mangleError  bool
	unsubscribed bool
	unsubClosed bool
}

func initSimpleSubscriber() (out *simpleSubscriber) {
	out = new(simpleSubscriber)
	out.out = make(chan Notification)
	out.unsub = make(chan interface{})
	return
}

func (sub *simpleSubscriber) Events() <-chan Notification {
	return sub.out
}

func (sub *simpleSubscriber) IsSubscribed() bool {
	return !sub.unsubscribed
}

func (sub *simpleSubscriber) Unsubscribe() {
	sub.lock.RLock()
	if !sub.IsSubscribed() {
		return
	}
	close(sub.unsub)
	sub.unsubClosed = true
	select {
	case sub.out <- Complete():
	default:
	}
	sub.lock.RUnlock()
	sub.lock.Lock()
	defer sub.lock.Unlock()
	sub.handleComplete()
}

func (sub *simpleSubscriber) Notify(n Notification) {
	sub.lock.RLock()
	if !sub.IsSubscribed() {
		return
	}
	if !sub.rawSend(n) {
		sub.lock.RUnlock()
		return
	}
	if n.Type == OnError && sub.mangleError {
		n = Complete()
		if !sub.rawSend(n) {
			sub.lock.RUnlock()
			return
		}
	}
	sub.lock.RUnlock()
	if n.Type == OnComplete {
		sub.lock.Lock()
		sub.handleComplete()
		sub.lock.Unlock()
	}
}

func (sub *simpleSubscriber) rawSend(n Notification) bool {
	select {
	case sub.out <- n:
		return true
	case <-sub.unsub:
		return false
	}
}

func (sub *simpleSubscriber) handleComplete() {
	if !sub.IsSubscribed() {
		return
	}
	sub.unsubscribed = true
	close(sub.out)
	if !sub.unsubClosed {
		close(sub.unsub)
		sub.unsubClosed = true
	}
	sub.callHooks()
}
