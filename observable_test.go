package urx

import (
	"testing"
	"time"
	"sync"
	"fmt"
)

func TestObservableBasic(t *testing.T) {
	obs := Create(func(sub Subscriber) {
		for i := int(0); i < 5; i++ {
			<-time.After(time.Millisecond * 25)
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
		for i := int(0); to < 0 || i < to; i++ {
			<-time.After(rate)
			inChan <- i
		}
		close(inChan)
	}()
	return obs.Publish()
}

func TestObservableFromChan(t *testing.T) {
	verifyObs(t, createChanObs(5, time.Millisecond * 25))
}

func TestObservablePublish(t *testing.T) {
	o := createChanObs(5, time.Millisecond * 25).Publish()
	var wg sync.WaitGroup
	verify := func() {
		defer wg.Done()
		i := verifyObs(t, o)
		if i != 5 {
			t.Error("the stream closed prematurely")
		}
	}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go verify()
	}
	wg.Wait()
}

func TestUnsubscribe(t *testing.T) {
	obs := createChanObs(5, time.Millisecond * 25).Publish()
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
}

func TestError(t *testing.T) {
	subj := NewPublishSubject()
	wait := make(chan interface{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		val := subj.Subscribe().Values()
		count := 0
		<-wait
		for range val {
			count++
		}
		if count != 1 {
			panic("error did not stop the stream")
		}
	}()
	wait <- nil
	subj.Next(nil)
	subj.Error(fmt.Errorf("test"))
	wg.Wait()
}

func verifyObs(t *testing.T, obs Observable) int {
	subscription := obs.Subscribe()
	hadStart, hadEnd := false, false
	i := 0
	for event := range subscription.Events() {
		switch event.Type {
		case OnStart:
			hadStart = true
		case OnComplete:
			hadEnd = true
		case OnNext:
			val := event.Body.(int)
			if val != i {
				t.Errorf("expecting %d but got %d", i, val)
				panic("invalid data through pipeline")
			}
			i++
		case OnError:
			panic(event.Body)
		default:
			panic(fmt.Sprintf("INVALID NOTIFICATION TYPE: %s", event.Type))
		}
	}
	if !hadStart {
		panic("did not get a Start()")
	}
	if !hadEnd {
		panic("did not get an End()")
	}
	return i
}

func BenchmarkObservableChannel(b *testing.B) {
	sub := createChanObs(-1, time.Duration(0)).Subscribe()
	for i := 0; i < b.N; i++ {
		<-sub.Events()
	}
	sub.Unsubscribe()
}

func BenchmarkObservableSimple(b *testing.B) {
	o := Create(func(sub Subscriber) {
		for i := 0; i < b.N; i++ {
			if !sub.IsSubscribed() {
				return
			}
			sub.Notify(Next(i))
		}
		sub.Notify(Complete())
	})
	s := o.Subscribe()
	for range s.Events() {}
}

func benchPubObs(goRoutineCount int, b *testing.B) {
	ready := make(chan interface{})
	obs := Create(func(sub Subscriber) {
		rCount := 0
		for rCount < goRoutineCount {
			<-ready
			rCount++
		}

		b.StopTimer()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			sub.Notify(Next(i))
			b.StopTimer()
		}
		b.StartTimer()
		sub.Notify(Complete())
		b.StopTimer()
	}).Publish()

	var wg sync.WaitGroup
	wg.Add(goRoutineCount)
	for i := 0; i < goRoutineCount; i++ {
		go func(finish int) {
			defer wg.Done()
			sub := obs.Subscribe()
			ready <- nil
			lastI := 0
			for i := range sub.Values() {
				iVal := i.(int)
				if iVal != lastI {
					panic(fmt.Sprintf("invalid value flowing down pipe: %d", i))
				}
				lastI++
			}
		}((goRoutineCount - i) / 2)
	}
	wg.Wait()
}

func BenchmarkPublishedObservable10(b *testing.B) {
	benchPubObs(10, b)
}

func BenchmarkPublishedObservable25(b *testing.B) {
	benchPubObs(25, b)
}

func BenchmarkPublishedObservable50(b *testing.B) {
	benchPubObs(50, b)
}

func BenchmarkPublishedObservable125(b *testing.B) {
	benchPubObs(125, b)
}