package urx

import "sync"

type CompositeSubscriptionTarget interface {
	IsSubscribed() bool
	Unsubscribe()
}

type CompositeSubscription struct {
	mutex sync.RWMutex
	closed bool
	hooks
}

func (sub *CompositeSubscription) Add(subscription CompositeSubscriptionTarget) {
	sub.mutex.Lock()
	defer sub.mutex.Lock()
	sub.hooks.Add(func() {
		if subscription.IsSubscribed() {
			subscription.Unsubscribe()
		}
	})
}

func (sub *CompositeSubscription) IsSubscribed() bool {
	sub.mutex.RLock()
	defer sub.mutex.RUnlock()
	return !sub.finished
}

func (sub *CompositeSubscription) Unsubscribe() {
	sub.mutex.Lock()
	defer sub.mutex.Lock()
	if sub.finished {
		return
	}
	sub.callHooks()
}
