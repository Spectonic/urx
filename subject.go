package urx

type simpleSubject struct {
	source chan Notification
	obs Observable
}

func NewPublishSubject() Subject {
	var out simpleSubject
	out.source = make(chan Notification)
	out.obs = Create(func (subscriber Subscriber) {
		for n := range out.source {
			subscriber.Notify(n)
			if n.Type == OnComplete {
				close(out.source)
			}
		}
	}).Publish()
	return out
}

func (s simpleSubject) Next(data interface{}) {
	s.source <- Next(data)
}

func (s simpleSubject) Error(err error) {
	s.source <- Error(err)
}

func (s simpleSubject) Complete() {
	s.source <- Complete()
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

func (s simpleSubject) Lift(o Operator) Subject {
	return simpleSubject{source: s.source, obs: s.obs.Lift(o)}
}