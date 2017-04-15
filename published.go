package urx

import (
	"sync"
	"math/rand"
)

type publishedObservable struct {
	completed   bool
	source      privObservable
	sub         privSubscription
	targetMutex sync.RWMutex
	targets     map[*simpleSubscriber]*simpleSubscriber
}

func (obs *publishedObservable) privSubscribe() privSubscription {
	obs.targetMutex.Lock()
	if obs.completed {
		newTarget := initSimpleSubscriber()
		go func() {
			newTarget.Notify(Start())
			newTarget.Notify(Complete())
		}()
		return newTarget
	}
	obs.initSubIfNeeded()

	newTarget := initSimpleSubscriber()
	newTarget.extraLockers = append(newTarget.extraLockers, &obs.targetMutex)
	obs.targets[newTarget] = newTarget
	go newTarget.Notify(Start())
	newTarget.Add(obs.removeTargetHook(newTarget))
	obs.targetMutex.Unlock()
	return newTarget
}

func (obs *publishedObservable) Unsubscribe() {
	if !obs.IsSubscribed() {
		return
	}
	obs.sub.Unsubscribe()
}

func (obs *publishedObservable) IsSubscribed() bool {
	return obs.sub != nil && obs.sub.IsSubscribed()
}

func (obs *publishedObservable) Add(h CompleteHook) {
	if obs.sub == nil {
		panic("cannot add while not subscribed")
	}

	obs.sub.Add(h)
}

func (obs *publishedObservable) initSubIfNeeded() {
	if obs.sub == nil {
		obs.sub = obs.source.privSubscribe()
		go obs.pump()
	}
}

func (obs *publishedObservable) Lift(op Operator) privObservable {
	return &liftedObservable{source: obs, op: op}
}

func (obs *publishedObservable) pump() {
	for e := range obs.sub.Events() {
		if  e.Type == OnStart {
			continue
		}
		obs.pumpNotification(e)
		if e.Type == OnComplete {
			obs.completed = true
			break
		}
	}
	obs.sub.Unsubscribe()
	obs.sub = nil
}

func (obs *publishedObservable) pumpNotification(n Notification) {
	obs.targetMutex.RLock()
	//this is designed this way such that we can send notifications to all listeners as they become ready
	//first, create all the select cases
	targets := make([]*simpleSubscriber, 0, len(obs.targets))

	//then create a reflect.Value from the notification we're sending
	//go through all the targets
	for i := range obs.targets {
		//and for this target
		target := obs.targets[i]
		target.RLock()
		if target.IsSubscribed() {
			//add it to our targets
			targets = append(targets, target)
		} else {
			target.RUnlock()
		}
	}
	obs.targetMutex.RUnlock()
	//and then iterate through select cases until we're done
	for len(targets) > 0 {
		//get a random target
		i := rand.Intn(len(targets))
		target := targets[i]
		//remove it from the targets collection
		targets = append(targets[:i], targets[i + 1:]...)
		//select from (send, unsub, none-ready)
		var unsubbed bool
		select {
		case target.out <- n:
		case <-target.unsub:
			unsubbed = true
		default:
			//if nothing is ready, re-prepare the target for later, and move on
			targets = append(targets, target)
			continue
		}
		//we reach here in every case except when we could not send or read unsub (default)
		//we're done with the lock
		target.RUnlock()
		//if it's an OnComplete, we're ready to call our hooks and shut down
		if n.Type == OnComplete && !unsubbed {
			target.Lock()
			target.handleComplete()
			target.Unlock()
		}
	}
}

func (obs *publishedObservable) removeTargetHook(target *simpleSubscriber) func() {
	return func() {
		delete(obs.targets, target)
		if len(obs.targets) == 0 {
			obs.Unsubscribe()
		}
	}
}
