package urx

type wrappedSubscription struct {
	source *publishedObservable
	composite *CompositeSubscription
	h hooks
}

func wrapSub(sub privObservable) (out wrappedSubscription) {
	if pubSub, ok := sub.(*publishedObservable); ok {
		out.source = pubSub
	} else {
		out.source = published(sub).(*publishedObservable)
	}
	out.composite = new(CompositeSubscription)
	return
}

func (s wrappedSubscription) Events() <- chan Notification {
	return s.subNew().Events()
}

func (s wrappedSubscription) subNew() privSubscription {
	sub := s.source.privSubscribe()
	s.composite.Add(sub)
	return sub
}

func (s wrappedSubscription) Unsubscribe() {
	s.composite.Unsubscribe()
	s.h.callHooks()
}

func (s wrappedSubscription) Values() <- chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for e := range s.Events() {
			if e.Type == OnNext {
				out <- e.Body
			}
		}
	}()
	return out
}

func (s wrappedSubscription) Error() <- chan error {
	out := make(chan error)
	go func() {
		defer close(out)
		for e := range s.Events() {
			if e.Type == OnError {
				out <- e.Body.(error)
				return
			}
		}
	}()
	return out
}

func (s wrappedSubscription) Complete() <- chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for e := range s.Events() {
			if e.Type == OnComplete {
				out <- nil
			}
		}
	}()
	return out
}

func (s wrappedSubscription) IsSubscribed() bool {
	return s.composite.IsSubscribed() && !s.h.finished
}

func (s wrappedSubscription) Add(hook CompleteHook) {
	s.h.Add(hook)
}
