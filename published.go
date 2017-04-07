package urx

import (
	"sync"
	"reflect"
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
	sendCases, unsubCases, targets := make([]reflect.SelectCase, 0, len(obs.targets)),
		make([]reflect.SelectCase, 0, len(obs.targets)),
		make([]*simpleSubscriber, 0, len(obs.targets))

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
		target.lock.RLock()
		if !target.IsSubscribed() {
			continue
		}
		//first, lock it so that we can use it
		addCase(target, send)
	}
	obs.targetMutex.RUnlock()
	cases := make([]reflect.SelectCase, 0, len(sendCases) + len(unsubCases))
	cases = append(cases, sendCases...)
	cases = append(cases, unsubCases...)
	del := func(targetIndexes ...int) {
		for i := range targetIndexes {
			targetIndex := targetIndexes[i]
			cases = append(cases[:targetIndex + len(sendCases)], cases[targetIndex + len(sendCases) + 1:]...)
			cases = append(cases[:targetIndex], cases[targetIndex + 1:]...)
			sendCases = append(sendCases[:targetIndex], sendCases[targetIndex + 1:]...)
			unsubCases = append(unsubCases[:targetIndex], unsubCases[targetIndex + 1:]...)
			targets = append(targets[:targetIndex], targets[targetIndex + 1:]...)
		}
	}
	//and then iterate through select cases until we're done
	for {
		//do some cleaning
		for i := 0; i < len(targets); i++ {
			if !targets[i].IsSubscribed() {
				del(i)
				i--
			}
		}
		if len(cases) == 0 {
			break
		}
		//we have this here so that if someone unsubscribes, we hand control over to the unsubscribe method
		//select on any of those things
		sent, _, _ := reflect.Select(cases)
		//if we are in the "sendCases" region
		if sent < len(sendCases) {
			target := targets[sent]
			sentN := sendCases[sent].Send.Interface().(Notification)
			del(sent)
			target.lock.RUnlock()
			//if we have an error, we should also send a complete
			if sentN.Type == OnComplete { //if this send isn't an error
				// we're done with the lock
				//if it's an OnComplete, we're also ready to call our hooks and shut down
				target.lock.Lock()
				target.handleComplete()
				target.lock.Unlock()
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
		obs.targetMutex.Lock()
		defer obs.targetMutex.Unlock()
		delete(obs.targets, target)
	}
}
