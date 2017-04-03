package urx

import (
	"testing"
	"time"
	"sync"
	"fmt"
)

func TestFilter(t *testing.T) {
	sub := createChanObs(10, time.Millisecond * 500).Filter(func (in interface{}) bool {
		return in.(int) % 2 == 0
	}).Subscribe()

	for i := range sub.Values() {
		if i.(int) % 2 != 0 {
			t.Errorf("got %d", i.(int))
			panic("the filter was ignored")
		}
		fmt.Printf("we got %d\n", i.(int))
	}
	fmt.Println("filter worked perfectly!")
}

func TestMap(t *testing.T) {
	sub := createChanObs(10, time.Millisecond * 500).Map(func (in interface{}) interface{} {
		return in.(int) * 2 + 20
	}).Subscribe()
	for i := range sub.Values() {
		val := i.(int)
		if val < 20 || val % 2 != 0 {
			t.Errorf("got %d", val)
			panic("the map failed")
		}
		fmt.Printf("we got %d\n", val)
	}
	fmt.Println("map worked perfectly")
}

//little helper thing
func Impulse() (o Observable, f func()) {
	c := make(chan interface{})
	o = FromChan(c).Publish()
	f = func() {
		c <- nil
		close(c)
	}
	return
}

func TestBuffer(t *testing.T) {
	await, activate := Impulse()
	in := 0
	sub := createChanObs(7, time.Second).Lift(FunctionOperator(func(s Subscriber, n Notification) {
		if n.Type == OnNext {
			fmt.Printf("notification %s came in at %s\n", n.Body, time.Now().Format("15:04:05 MST"))
			in++
			if in == 5 {
				activate()
			}
		}
		s.Notify(n)
	})).Buffered(3).Subscribe()

	<-await.Subscribe().Complete()
	for i := range sub.Values() {
		val := i.(int)
		fmt.Printf("got %d in main loop at %s\n", val, time.Now().Format("15:04:05 MST"))
	}
	fmt.Println("finished, got all values")
}

func TestMultipleOperators(t *testing.T) {
	obs := createChanObs(20, time.Millisecond * 50).Map(func (in interface{}) interface{} {
		return (in.(int) * 10) + 12
	}).Filter(func (in interface{}) bool {
		return in.(int) % 3 == 0
	}).Publish()

	var wg sync.WaitGroup
	validate := func() {
		defer wg.Done()
		for i := range obs.Subscribe().Values() {
			val := i.(int)
			if val % 3 != 0 || val < 12 {
				t.Errorf("invalid value supplied %d", val)
				panic(fmt.Sprintf("operators failed: %d", val))
			}
			fmt.Printf("we got %d\n", val)
		}
		fmt.Println("one subscriber finished!")
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go validate()
	}
	wg.Wait()
	fmt.Println("all subscribers finished!")
}