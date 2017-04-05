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
	//used to write up to a parent when an unsubscription
	hooks
	lock sync.RWMutex
	mangleError bool
}

func initSimpleSubscriber() (out *simpleSubscriber) {
	out = new(simpleSubscriber)
	out.out = make(chan Notification)
	return
}

func (sub *simpleSubscriber) Events() <-chan Notification {
	return sub.out
}

func (sub *simpleSubscriber) Unsubscribe() {
	sub.lock.RLock()
	if !sub.IsSubscribed() {
		return
	}
	sub.out <- Complete()
	sub.lock.RUnlock()
	sub.handleComplete()
}

func (sub *simpleSubscriber) Notify(n Notification) {
	sub.lock.RLock()
	if !sub.IsSubscribed() {
		return
	}
	sub.out <- n
	if n.Type == OnError && sub.mangleError {
		n = Complete()
		sub.out <- n
	}
	sub.lock.RUnlock()
	if n.Type == OnComplete {
		sub.handleComplete()
	}
}

func (sub *simpleSubscriber) handleComplete() {
	sub.lock.Lock()
	defer sub.lock.Unlock()
	if !sub.IsSubscribed() {
		return
	}
	close(sub.out)
	sub.callHooks()
}

type liftedObservable struct {
	source privObservable
	op     Operator
}

type liftedSubscriber struct {
	source privSubscription
	op     Operator
	events chan Notification
	hooks
}

func (lifted *liftedObservable) privSubscribe() (sub privSubscription) {
	out := &liftedSubscriber{source: lifted.source.privSubscribe(), op: lifted.op, events: make(chan Notification)}
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

func (sub *liftedSubscriber) Events() <-chan Notification {
	return sub.events
}

func (sub *liftedSubscriber) Unsubscribe() {
	sub.source.Unsubscribe()
}

func (sub *liftedSubscriber) Notify(not Notification) {
	sub.events <- not
	if not.Type == OnComplete {
		close(sub.events)
		sub.callHooks()
	}
}

type publishedObservable struct {
	source      privObservable
	sub         privSubscription
	targetMutex sync.RWMutex
	targets     map[*simpleSubscriber]*simpleSubscriber
}

func (obs *publishedObservable) privSubscribe() privSubscription {
	obs.targetMutex.Lock()
	if obs.sub == nil {
		obs.sub = obs.source.privSubscribe()
		go obs.pump()
	}

	newTarget := initSimpleSubscriber()
	obs.targets[newTarget] = newTarget
	go newTarget.Notify(Start())
	obs.targetMutex.Unlock()
	newTarget.Add(obs.removeTargetHook(newTarget))
	return newTarget
}

func (obs *publishedObservable) Lift(op Operator) privObservable {
	return &liftedObservable{source: obs, op: op}
}

func (obs *publishedObservable) pump() {
	for e := range obs.sub.Events() {
		if e.Type != OnNext && e.Type != OnError {
			continue
		}
		obs.targetMutex.RLock()
		for target := range obs.targets {
			target.Notify(e)
		}
		obs.targetMutex.RUnlock()
	}
	obs.targetMutex.RLock()
	for target := range obs.targets {
		target.Notify(Complete())
	}
	obs.targetMutex.RUnlock()
}

func (obs *publishedObservable) removeTargetHook(target *simpleSubscriber) func() {
	return func() {
		delete(obs.targets, target)
	}
}

//hooks util
type hooks struct {
	slice    []CompleteHook
	m        sync.Mutex
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

func (h *hooks) IsSubscribed() bool {
	return !h.finished
}
