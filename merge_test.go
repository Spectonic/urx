package urx

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

const count = 2

func TestMerge(t *testing.T) {
	one := createChanObs(10, time.Millisecond*50).Map(func(in interface{}) interface{} {
		return in.(int) * -1
	}).Publish()

	two := createChanObs(20, time.Millisecond*25).Publish()

	var wg sync.WaitGroup
	values := make([][]int, count, count)
	wg.Add(count)
	for i := 0; i < count; i++ {
		s := &values[i]
		*s = make([]int, 0, 0)
		go func() {
			defer wg.Done()
			for i := range Merge(one, two).Subscribe().Values() {
				*s = append(*s, i.(int))
			}
		}()
	}
	wg.Wait()

	for i := range values {
		for i0 := range values {
			if i0 == i {
				continue
			}
			if len(values[i0]) != len(values[i]) || len(values[i0]) != 30 {
				fmt.Printf("[%d] %v\n [%d] %v", len(values[i0]), values[i0], len(values[i]), values[i])
				panic("non-equal slices")
			}
		}
	}
}
