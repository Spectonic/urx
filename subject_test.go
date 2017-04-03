package urx

import (
	"testing"
	"sync"
	"time"
	"fmt"
)

func TestSubject(t *testing.T) {
	const numsSent = 10
	subj := NewPublishSubject()
	var wg sync.WaitGroup

	var waiting []Observable
	for i := 0; i < 150; i++ {
		obs, trigger := Impulse()
		wg.Add(1)
		go func() {
			defer wg.Done()
			x := 0
			sub := subj.Subscribe()
			c := sub.Values()
			trigger()
			for n := range c {
				x = n.(int)
			}
			if x != numsSent - 1 {
				panic(fmt.Sprintf("failed to get %d more items via %p", numsSent - x - 1, c))
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
