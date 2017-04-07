package urx

import (
	"sync"
	"math/rand"
)

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
	newTarget.extraLockers = append(newTarget.extraLockers, &obs.targetMutex)
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
		obs.pumpNotification(e)
	}
	obs.pumpNotification(Complete())
}

func (obs *publishedObservable) pumpNotification(n Notification) {
	obs.targetMutex.RLock()
	//this is designed this way such that we can send notifications to all listeners as they become ready
	//first, create all the select cases
	sendCases, unsubCases, targets := make([]chan Notification, 0, len(obs.targets)),
		make([]chan interface{}, 0, len(obs.targets)),
		make([]*simpleSubscriber, 0, len(obs.targets))
	addCase := func(target *simpleSubscriber) {
		//add it to our targets
		targets = append(targets, target)
		//and add it to our select cases
		sendCases = append(sendCases, target.out)
		unsubCases = append(unsubCases, target.unsub)
	}

	//then create a reflect.Value from the notification we're sending
	//go through all the targets
	for i := range obs.targets {
		//and for this target
		target := obs.targets[i]
		target.RLock()
		if !target.IsSubscribed() {
			continue
		}
		//first, lock it so that we can use it
		addCase(target)
	}
	obs.targetMutex.RUnlock()
	del := func(targetIndexes ...int) {
		for i := range targetIndexes {
			targetIndex := targetIndexes[i]
			sendCases = append(sendCases[:targetIndex], sendCases[targetIndex + 1:]...)
			unsubCases = append(unsubCases[:targetIndex], unsubCases[targetIndex + 1:]...)
			targets = append(targets[:targetIndex], targets[targetIndex + 1:]...)
		}
	}
	//and then iterate through select cases until we're done
	for len(targets) > 0 {
		i := rand.Intn(len(unsubCases))
		target := targets[i]
		cleanup := func() {
			del(i)
			target.RUnlock()
		}
		select {
		case sendCases[i] <- n:
			cleanup()
			//if we have an error, we should also send a complete
			if n.Type == OnComplete { //if this send isn't an error
				// we're done with the lock
				//if it's an OnComplete, we're also ready to call our hooks and shut down
				target.Lock()
				target.handleComplete()
				target.Unlock()
			}
		case <-unsubCases[i]:
			cleanup()
		default:
		}
	}
}

func (obs *publishedObservable) removeTargetHook(target *simpleSubscriber) func() {
	return func() {
		delete(obs.targets, target)
	}
}
