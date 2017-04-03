package urx

// creates an observable from a function
func Create(onSub func(Subscriber)) Observable {
	return wrapObservable(simpleObservable(onSub))
}


// creates a published observable from an observable
func published(source privObservable) privObservable {
	s := source.Subscribe()
	out := &publishedObservable{source: s, targets: make(map[*simpleSubscriber]*simpleSubscriber)}
	go out.pump()
	return out
}

type Operator interface {
	Notify(Subscriber, Notification)
}

// The generic observable interface is what fundamentally defines an observable
// an observable can be subscribed to, and can be used to create derived observables
type privObservable interface {
	Subscribe() Subscription
	Lift(Operator) privObservable
}

type privObserver interface {
	Notify(Notification)
}

type privSubject interface {
	privObservable
	Subscriber
	Next(interface{}) error
}


type Subscriber interface {
	privObserver
	Add(CompleteHook)
}

type CompleteHook func()

type Subscription interface {
	Events() <- chan Notification
	Unsubscribe()
}