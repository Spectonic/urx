package urx

type wrappedSubscription struct {
	source privSubscription
}

func (s wrappedSubscription) Events() <- chan Notification {
	return s.source.Events()
}

func (s wrappedSubscription) Unsubscribe() {
	s.source.Unsubscribe()
}

func (s wrappedSubscription) Values() <- chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for e := range s.source.Events() {
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
		for e := range s.source.Events() {
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
		for e := range s.source.Events() {
			if e.Type == OnComplete {
				out <- nil
			}
		}
	}()
	return out
}

func (s wrappedSubscription) IsSubscribed() bool {
	return s.source.IsSubscribed()
}

func (s wrappedSubscription) Add(hook CompleteHook) {
	s.source.Add(hook)
}
