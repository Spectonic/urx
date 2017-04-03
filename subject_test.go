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
	for i := 0; i < 10; i++ {
		obs, trigger := Impulse()
		wg.Add(1)
		go func() {
			defer wg.Done()
			x := 0
			sub := subj.Subscribe()
			c := sub.Values()
			trigger()
			fmt.Printf("ready\n")
			for n := range c {
				x = n.(int)
				fmt.Printf("got %d via %p\n", x, c)
				<-time.After(time.Second)
			}
			if x != numsSent - 1 {
				panic(fmt.Sprintf("failed to get %d more items via %p", numsSent - x - 1, c))
			}
		}()
		waiting = append(waiting, obs)
	}
	<-Merge(waiting...).Subscribe().Complete()
	fmt.Printf("firing\n")
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numsSent; i++ {
			subj.Next(i)
			<-time.After(time.Millisecond * 150)
		}
		subj.Complete()
	}()

	wg.Wait()
}
