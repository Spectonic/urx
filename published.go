package urx

import (
	"sync"
	"reflect"
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
	sendCases, unsubCases, targets := make([]reflect.SelectCase, 0, len(obs.targets)),
		make([]reflect.SelectCase, 0, len(obs.targets)),
		make([]*simpleSubscriber, 0, len(obs.targets))
	obs.targetMutex.RUnlock()

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
	obs.targetMutex.RLock()
	for i := range obs.targets {
		//and for this target
		target := obs.targets[i]
		target.RLock()
		if !target.IsSubscribed() {
			continue
		}
		//first, lock it so that we can use it
		addCase(target, send)
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
	cases := make([]reflect.SelectCase, 3, 3)
	//and then iterate through select cases until we're done
	for {
		var i int

	select_target:
		if len(targets) == 0 {
			break
		}
		i = rand.Intn(len(unsubCases))
		target := targets[i]
		if !target.IsSubscribed() {
			del(i)
			goto select_target
		}

		cases[0], cases[1], cases[2] = sendCases[i], unsubCases[i], reflect.SelectCase{Dir: reflect.SelectDefault}
		//we have this here so that if someone unsubscribes, we hand control over to the unsubscribe method
		//select on any of those things
		sent, _, _ := reflect.Select(cases)
		//if we are in the "sendCases" region
		if sent == 0 {
			sentN := cases[0].Send.Interface().(Notification)
			del(i)
			target.RUnlock()
			//if we have an error, we should also send a complete
			if sentN.Type == OnComplete { //if this send isn't an error
				// we're done with the lock
				//if it's an OnComplete, we're also ready to call our hooks and shut down
				target.Lock()
				target.handleComplete()
				target.Unlock()
			}
		} else if sent == 1 {
			//if someone unsubscribes, we simply RUnlock and hand control over to the Unsubscribe method
			sent -= len(sendCases)
			targets[sent].RUnlock()
			del(i)
		}
	}
}

func (obs *publishedObservable) removeTargetHook(target *simpleSubscriber) func() {
	return func() {
		delete(obs.targets, target)
	}
}
