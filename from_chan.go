package urx

import "reflect"

func FromChan(source interface{}) Observable {
	val := reflect.ValueOf(source)
	if val.Kind() != reflect.Chan {
		panic("a channel was not passed to urx.FromChan")
	}

	sub := func(sub Subscriber) {
		for {
			next, ok := val.Recv()
			if !ok {
				sub.Notify(Notification{Body: nil, Type: OnComplete})
				return
			} else {
				sub.Notify(Notification{Body: next.Interface(), Type: OnNext})
			}
		}
	}

	out := simpleObservable{&sub}
	return bObservable{out}
}
