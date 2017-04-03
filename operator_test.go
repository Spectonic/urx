package urx

import (
	"testing"
	"time"
)

func TestFilter(t *testing.T) {
	sub := createChanObs(10, time.Millisecond * 5).Filter(func (in interface{}) bool {
		return in.(int) % 2 == 0
	}).Subscribe()
	for i := range sub.Values() {
		if i.(int) % 2 != 0 {
			t.Errorf("got %d", i.(int))
			panic("the filter was ignored")
		}
	}
	t.Log("filter worked perfectly!")
}