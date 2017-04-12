package urx

// creates an observable from a function
func Create(onSub func(Subscriber)) Observable {
	return bObservable{simpleObservable{&onSub}}
}

// creates a published observable from an observable
func published(source privObservable) *publishedObservable {
	return &publishedObservable{source: source, targets: make(map[*simpleSubscriber]*simpleSubscriber)}
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
	RootSubscriber
}

type Observer interface {
	Notify(Notification)
}

type CompleteHook func()

type Subscriber interface {
	Observer
	RootSubscriber
}

type Subject interface {
	Next(interface{})
	Error(error)
	Complete()
	Post(Notification)
	Subscribe() Subscription
	AsObservable() PublishedObservable
}

type RootSubscriber interface {
	IsSubscribed() bool
	Add(CompleteHook)
}