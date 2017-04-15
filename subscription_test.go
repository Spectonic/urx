package urx

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSubscription(t *testing.T) {
	obs := createChanObs(10, time.Millisecond*25).Publish()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(take int) {
			defer wg.Done()

			handled := false
			sub := obs.Subscribe()
			sub.Add(func() {
				handled = true
			})
			var last int
			for value := range sub.Values() {
				last = value.(int)
				if take == last {
					sub.Unsubscribe()
					//should naturally cause the channel to close
				}
			}

			if last != take {
				panic(fmt.Sprintf("invalid situation, did not read correct number of items: %d != %d", last, take))
			}

			if sub.IsSubscribed() {
				panic("subscription that is unsubscribed is incorrectly reporting as subscribed")
			}

			if !handled {
				panic("subscription did not call hoook!")
			}
		}(i)
	}
	wg.Wait()
}
