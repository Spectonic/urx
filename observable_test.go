package urx

import (
	"testing"
	"time"
	"sync"
)

func TestObservableBasic(t *testing.T) {
	obs := Create(func(sub Subscriber) {
		for i := int(0); i < 5; i++ {
			<-time.After(time.Second)
			sub.Notify(Next(i))
		}
		sub.Notify(Complete())
	})

	verifyObs(t, obs)
}

func createChanObs(to int, rate time.Duration) Observable {
	inChan := make(chan int)
	obs := FromChan(inChan)
	go func() {
		for i := int(0); i < to; i++ {
			<-time.After(rate)
			inChan <- i
		}
		close(inChan)
	}()
	return obs
}

func TestObservableFromChan(t *testing.T) {
	verifyObs(t, createChanObs(5, time.Second))
}

func TestObservablePublish(t *testing.T) {
	o := createChanObs(5, time.Second).Publish()
	var wg sync.WaitGroup
	verify := func() {
		defer wg.Done()
		i := verifyObs(t, o)
		if i != 5 {
			t.Error("the stream closed prematurely")
		}
	}
	wg.Add(2)
	go verify()
	go verify()
	wg.Wait()
}

func TestUnsubscribe(t *testing.T) {
	obs := createChanObs(5, time.Second).Publish()
	root := func() {
		sub := obs.Subscribe()
		for i := 0; i < 2; i++ {
			<-sub.Values()
		}
		sub.Unsubscribe()
	}
	var wg sync.WaitGroup
	other := func() {
		defer wg.Done()
		sub := obs.Subscribe()
		i := 0
		for range sub.Values() {
			i++
		}
		if i != 5 {
			t.Errorf("only got %d values when expected 5", i)
			panic("unsubscribe changed result")
		}
	}
	wg.Add(1)
	go other()
	root()
	wg.Wait()
	t.Log("succeeded")
}

func BenchmarkObservableSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sub := createChanObs(5, time.Millisecond * 5).Subscribe()
		for range sub.Events() {}
	}
}

func verifyObs(t *testing.T, obs Observable) int {
	subscription := obs.Subscribe()
	i := 0
	for event := range subscription.Values() {
		val := event.(int)
		if val != i {
			t.Errorf("expecting %d but got %d", i, val)
			panic("invalid data through pipeline")
		}
		t.Logf("got %d at %s", val, time.Now().Format("15:04:05 MST"))
		i++
	}
	t.Log("got all values, stream closed")
	return i
}