package urx

type CompositeSubscription struct {
	closed bool
	hooks
}

func (sub *CompositeSubscription) Add(subscription Subscription) {
	sub.hooks.Add(func() {
		if subscription.IsSubscribed() {
			subscription.Unsubscribe()
		}
	})
}

func (sub *CompositeSubscription) IsSubscribed() bool {
	return !sub.finished
}

func (sub *CompositeSubscription) Unsubscribe() {
	sub.callHooks()
}
