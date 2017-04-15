package urx

type simpleSubject struct {
	source chan Notification
	obs    PublishedObservable
}

func NewPublishSubject() Subject {
	var out simpleSubject
	out.source = make(chan Notification)
	out.obs = Create(func(subscriber Subscriber) {
		for n := range out.source {
			subscriber.Notify(n)
			if n.Type == OnComplete {
				close(out.source)
			}
		}
	}).Publish()
	out.obs.getObs().(*publishedObservable).initSubIfNeeded()
	return out
}

func (s simpleSubject) Next(data interface{}) {
	s.Post(Next(data))
}

func (s simpleSubject) Error(err error) {
	s.Post(Error(err))
}

func (s simpleSubject) Complete() {
	s.Post(Complete())
}

func (s simpleSubject) Post(n Notification) {
	s.source <- n
}

func (s simpleSubject) Subscribe() Subscription {
	return s.obs.Subscribe()
}

func (s simpleSubject) AsObservable() Observable {
	return s.obs
}
