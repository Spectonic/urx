package urx

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSubject(t *testing.T) {
	const numsSent = 100
	subj := NewPublishSubject()
	var wg sync.WaitGroup

	var waiting []Observable
	for i := 0; i < 150; i++ {
		obs, trigger := Impulse()
		wg.Add(1)
		go func() {
			defer wg.Done()
			x := 0
			got := 0
			sub := subj.Subscribe()
			c := sub.Values()
			trigger()
			for n := range c {
				x = n.(int)
				got++
			}
			if x != numsSent-1 {
				panic(fmt.Sprintf("failed to get %d more items via %p", numsSent-x-1, c))
			}
			if got != numsSent {
				panic("did not get enough numbers")
			}
		}()
		waiting = append(waiting, obs)
	}
	<-Merge(waiting...).Subscribe().Complete()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numsSent; i++ {
			subj.Next(i)
			<-time.After(time.Millisecond * 25)
		}
		subj.Complete()
	}()

	wg.Wait()
}
