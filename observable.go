package urx

import "sync"


// The simple observable is simply a function which takes a subscriber and provides it with data
type simpleObservable func(Subscriber)

// this creates a subscription (by calling the simpleObservable function immediately)
func (obs simpleObservable) Subscribe() Subscription {
	//first, create a subscriber/observer combo
	outSub := initSimpleSubscriber()
	go outSub.pump()
	obs(outSub)
	return outSub
}

// applies an operator to the observable such that subscriptions to the resulting observable flow through the operator
func (obs simpleObservable) Lift(op Operator) (newObs privObservable) {
	newObs = &liftedObservable{obs, op}
	return
}

type simpleSubscriber struct {
	//notifications from source are written here
	in chan Notification
	//then channel used to send notifications to subscribers
	out chan Notification
	//used to write up to a parent when an unsubscription
	unsub chan interface{}
	*hooks
}


func initSimpleSubscriber() (out *simpleSubscriber) {
	out = new(simpleSubscriber)
	out.in, out.out = make(chan Notification), make(chan Notification)
	out.unsub = make(chan interface{})
	return
}

func (sub *simpleSubscriber) Events() <-chan Notification {
	return sub.in
}

func (sub *simpleSubscriber) Unsubscribe() {
	sub.unsub <- nil
}

func (sub *simpleSubscriber) Notify(n Notification) {
	sub.out <- n
}

func (sub *simpleSubscriber) pump() {
	handleComplete := func() {
		sub.callHooks()
		close(sub.in)
		close(sub.out)
		close(sub.unsub)
	}

	for {
		select {
		case i := <-sub.in:
			sub.out <- i
			if i.t == OnComplete {
				handleComplete()
				return
			}
		case <-sub.unsub:
			handleComplete()
			return
		}
	}
}

type liftedObservable struct {
	source privObservable
	op     Operator
}

type liftedSubscriber struct {
	source Subscription
	op Operator
	out chan Notification
	*hooks
}

func (lifted *liftedObservable) Subscribe() (sub Subscription) {
	out := &liftedSubscriber{source: lifted.source.Subscribe(), op: lifted.op}
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
}

func (sub *liftedSubscriber) Events() <- chan Notification {
	return sub.out
}

func (sub *liftedSubscriber) Unsubscribe() {
	sub.source.Unsubscribe()
}

func (sub *liftedSubscriber) Notify(not Notification) {
	if not.t == OnComplete {
		sub.callHooks()
	}
	sub.out <- not
}

type publishedObservable struct {
	source Subscription
	targetMutex sync.RWMutex
	targets map[*simpleSubscriber]*simpleSubscriber
}

func (sub *publishedObservable) Subscribe() Subscription {
	newTarget := initSimpleSubscriber()
	sub.targetMutex.Lock()
	defer sub.targetMutex.Unlock()
	sub.targetMutex[newTarget] = newTarget
	newTarget.Add(sub.removeTarget(newTarget))
	return newTarget
}

func (sub *publishedObservable) Lift(op Operator) privObservable {
	return &liftedObservable{source: sub, op: op}
}

func (sub *publishedObservable) pump() {
	for e := range sub.source.Events() {
		sub.pumpNotification(e)
	}
}

func (sub *publishedObservable) pumpNotification(e Notification) {
	sub.targetMutex.RLock()
	defer sub.targetMutex.RUnlock()

	for target := range sub.targets {
		target.Notify(e)
	}
}

func (sub *publishedObservable) removeTarget(target *simpleSubscriber) func() {
	return func() {
		sub.targetMutex.Lock()
		defer sub.targetMutex.Unlock()

		delete(sub.targets, target)
	}
}

//hooks util
type hooks struct {
	slice []CompleteHook
	m sync.Mutex
	finished bool
}

func (h *hooks) Add(hook CompleteHook) {
	h.m.Lock()
	defer h.m.Unlock()

	if h.finished {
		return
	}

	h.slice = append(h.slice, hook)
	return
}

func (h *hooks) callHooks() {
	h.m.Lock()
	defer h.m.Unlock()

	for i := range h.slice {
		h.slice[i]()
	}
	h.slice = nil
	h.finished = true
}
