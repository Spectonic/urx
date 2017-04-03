package urx

import (
	"sync"
)

type Observable struct {
	privObservable privObservable
}

type Subscription interface {
	Events() <- chan Notification
	Unsubscribe()
	Values() <- chan interface{}
	Error() <- chan error
	Complete() <- chan interface{}
	IsSubscribed() bool
}

func (o Observable) Publish() Observable {
	return Observable{published(o.privObservable)}
}

func (o Observable) Lift(operator Operator) Observable {
	return Observable{o.privObservable.Lift(operator)}
}

func (o Observable) Map(m func (interface{}) interface{}) Observable {
	return o.Lift(FunctionOperator(func(sub Subscriber, n Notification) {
		if n.Type == OnNext {
			n.Body = m(n.Body)
		}
		sub.Notify(n)
	}))
}

func (o Observable) Filter(f func(interface{}) bool) Observable {
	return o.Lift(FunctionOperator(func(sub Subscriber, n Notification) {
		if n.Type == OnNext && !f(n.Body) {
			return
		}
		sub.Notify(n)
	}))
}

func (o Observable) Buffered(buffer int) Observable {
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

func (o Observable) Subscribe() Subscription {
	return wrappedSubscription{o.privObservable.privSubscribe()}
}

