package urx

// creates an observable from a function
func Create(onSub func(Subscriber)) Observable {
	return Observable{simpleObservable(onSub)}
}

// creates a published observable from an observable
func published(source privObservable) privObservable {
	s := source.privSubscribe()
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
	privSubscribe() privSubscription
	Lift(Operator) privObservable
}

type privSubscription interface {
	Events() <- chan Notification
	Unsubscribe()
	IsSubscribed() bool
}

type Observer interface {
	Notify(Notification)
}

type Subject interface {
	privObservable
	Subscriber
	Next(interface{}) error
}


type Subscriber interface {
	Observer
	Add(CompleteHook)
}

type CompleteHook func()