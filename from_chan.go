package urx

import "reflect"

func FromChan(source interface{}) Observable {
	val := reflect.ValueOf(source)
	if val.Kind() != reflect.Chan {
		panic("a channel was not passed to urx.FromChan")
	}

	out := simpleObservable(func(sub Subscriber) {
		open := true
		sub.Add(func() {
			if open {
				val.Close()
			}
		})
		for {
			next, ok := val.Recv()
			if !ok {
				open = false
				sub.Notify(Notification{body: nil, t: OnComplete})
				return
			} else {
				sub.Notify(Notification{body: next.Interface(), t: OnNext})
			}
		}
	})
	return wrapObservable(out)
}
