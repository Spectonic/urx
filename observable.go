package urx

import (
	"sync"
)

type FunctionOperator func(Subscriber, Notification)

func (o FunctionOperator) Notify(s Subscriber, n Notification) {
	o(s, n)
}

type Observable interface {
	Publish() PublishedObservable
	Lift(Operator) Observable
	Map(m func (interface{}) interface{}) Observable
	Filter(func(interface{}) bool) Observable
	Buffered(buffer int) Observable
	Subscribe() Subscription

	getObs() privObservable
}

type PublishedObservable interface {
	Observable
	Unsubscribe()
	IsSubscribed() bool
}

type pObservable struct {
	bObservable
}

func (p pObservable) IsSubscribed() bool {
	return p.privObservable.(*publishedObservable).IsSubscribed()
}

func (p pObservable) Unsubscribe() {
	p.privObservable.(*publishedObservable).Unsubscribe()
}

func (p pObservable) Publish() PublishedObservable {
	return p
}

type bObservable struct {
	privObservable privObservable
}

func (o bObservable) Publish() PublishedObservable {
	o.privObservable = published(o.privObservable)
	return pObservable{o}
}

func (o bObservable) Lift(operator Operator) Observable {
	return bObservable{o.privObservable.Lift(operator)}
}

func (o bObservable) Map(m func (interface{}) interface{}) Observable {
	return o.Lift(FunctionOperator(func(sub Subscriber, n Notification) {
		if n.Type == OnNext {
			n.Body = m(n.Body)
		}
		sub.Notify(n)
	}))
}

func (o bObservable) getObs() privObservable {
	return o.privObservable
}

func (o bObservable) Filter(f func(interface{}) bool) Observable {
	return o.Lift(FunctionOperator(func(sub Subscriber, n Notification) {
		if n.Type == OnNext && !f(n.Body) {
			return
		}
		sub.Notify(n)
	}))
}

func (o bObservable) Buffered(buffer int) Observable {
	type buffered struct {
		to Subscriber
		body Notification
	}

	var c chan buffered
	var l sync.RWMutex

	pump := func() {
		for msg := range c {
			msg.to.Notify(msg.body)
		}
	}

	start := func() {
		c = make(chan buffered, buffer)
		go pump()
	}

	return o.Lift(FunctionOperator(func(sub Subscriber, n Notification) {
		if n.Type == OnStart {
			l.Lock()
			l.Unlock()
			start()
		} else {
			l.RLock()
			defer l.RUnlock()
		}
		c <- buffered{sub, n}
		if n.Type == OnComplete {
			close(c)
		}
	}))
}

func (o bObservable) Subscribe() Subscription {
	return wrappedSubscription{sub: o.privObservable.privSubscribe()}.init()
}
