package urx

// creates an observable from a function
func Create(onSub func(Subscriber)) Observable {
	return Observable{simpleObservable{&onSub}}
}

// creates a published observable from an observable
func published(source privObservable) privObservable {
	out := &publishedObservable{source: source, targets: make(map[*simpleSubscriber]*simpleSubscriber)}
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
	Add(CompleteHook)
}

type Observer interface {
	Notify(Notification)
}

type CompleteHook func()

type Subscriber interface {
	Observer
	IsSubscribed() bool
	Add(CompleteHook)
}

type Subject interface {
	Next(interface{})
	Error(error)
	Complete()
	Post(Notification)
	Subscribe() Subscription
	Lift(Operator) Subject
	AsObservable() Observable
}