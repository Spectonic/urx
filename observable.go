package urx

import (
	"sync"
	"reflect"
	"fmt"
	"runtime/debug"
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
	lock sync.RWMutex
	mangleError bool
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

func (sub *simpleSubscriber) Unsubscribe() {
	sub.lock.RLock()
	if !sub.IsSubscribed() {
		return
	}
	select {
	case sub.unsub <- nil:
	default:
	}
	select {
	case sub.out <- Complete():
	default:
	}
	sub.lock.RUnlock()
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
		sub.handleComplete()
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
	sub.lock.Lock()
	defer sub.lock.Unlock()
	if !sub.IsSubscribed() {
		return
	}
	close(sub.out)
	close(sub.unsub)
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
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("debug: %+v -> %+v\n", *obs, e)
					debug.PrintStack()
					panic(r)
				}
			}()
			obs.pumpNotification(e)
		}()

	}
	obs.pumpNotification(Complete())
}

func (obs *publishedObservable) pumpNotification(n Notification) {
	obs.targetMutex.RLock()
	defer obs.targetMutex.RUnlock()

	//this is designed this way such that we can send notifications to all listeners as they become ready
	//first, create all the select cases
	var sendCases, unsubCases []reflect.SelectCase
	//and also a parallel collection of all the subscribers
	var targets []*simpleSubscriber

	addCase := func(target *simpleSubscriber, send reflect.Value) {
		//add it to our targets
		targets = append(targets, target)
		//and add it to our select cases
		sendCases = append(sendCases, reflect.SelectCase{
			Chan: reflect.ValueOf(target.out),
			Dir: reflect.SelectSend,
			Send: send})
		unsubCases = append(unsubCases, reflect.SelectCase{
			Chan: reflect.ValueOf(target.unsub),
			Dir: reflect.SelectRecv,
		})
	}

	//then create a reflect.Value from the notification we're sending
	send := reflect.ValueOf(n)
	//go through all the targets
	for i := range obs.targets {
		//and for this target
		target := obs.targets[i]
		//first, lock it so that we can use it
		target.lock.RLock()
		addCase(target, send)
	}
	//now, if we need it, create a complete notification value as well
	var completeN *reflect.Value
	if n.Type == OnError {
		c := reflect.ValueOf(Complete())
		completeN = &c
	}
	del := func(targetIndexes ...int) {
		for i := range targetIndexes {
			targetIndex := targetIndexes[i]
			sendCases = append(sendCases[:targetIndex], sendCases[targetIndex + 1:]...)
			unsubCases = append(unsubCases[:targetIndex], unsubCases[targetIndex + 1:]...)
			targets = append(targets[:targetIndex], targets[targetIndex+ 1:]...)
		}
	}
	//and then iterate through select cases until we're done
	for len(sendCases) > 0 {
		//do some cleaning
		var toDelete []int
		for i := range targets {
			if !targets[i].IsSubscribed() {
				toDelete = append(toDelete, i)
			}
		}
		del(toDelete...)
		//we have this here so that if someone unsubscribes, we hand control over to the unsubscribe method
		var cases []reflect.SelectCase
		//two regions of the "cases" slice
		cases = append(cases, sendCases...)
		cases = append(cases, unsubCases...)
		//select on any of those things
		sent, _, _ := reflect.Select(cases)
		//if we are in the "sendCases" region
		if sent < len(sendCases) {
			target := targets[sent]
			sentN := sendCases[sent].Send.Interface().(Notification)
			del(sent)
			//if we have an error, we should also send a complete
			if sentN.Type == OnError {
				//we send a complete by adding it to our current select cases
				addCase(target, *completeN)
			} else { //if this send isn't an error
				// we're done with the lock
				target.lock.RUnlock()
				//if it's an OnComplete, we're also ready to call our hooks and shut down
				if sentN.Type == OnComplete {
					target.handleComplete()
				}
			}
		} else {
			//if someone unsubscribes, we simply RUnlock and hand control over to the Unsubscribe method
			sent -= len(sendCases)
			targets[sent].lock.RUnlock()
			del(sent)
		}
	}
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
