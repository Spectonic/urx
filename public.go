package urx

func wrapObservable(observable privObservable) Observable {
	return Observable{observable}
}

type Observable struct {
	privObservable
}

func (o Observable) Publish() Observable {
	return wrapObservable(published(o.privObservable))
}

func (o Observable) Map(m func (interface{}) interface{}) Observable {
	return wrapObservable(o.Lift(FunctionOperator(func(sub Subscriber, n Notification) {
		if n.t == OnNext {
			n.body = m(n.body)
		}
		sub.Notify(n)
	})))
}

func (o Observable) Filter(f func(interface{}) bool) Observable {
	return wrapObservable(o.Lift(FunctionOperator(func(sub Subscriber, n Notification) {
		if n.t == OnNext && !f(n.body) {
			return
		}
		sub.Notify(n)
	})))
}