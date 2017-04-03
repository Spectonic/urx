package urx

import (
	"testing"
	"time"
)

func TestMerge(t *testing.T) {
	one := createChanObs(10, time.Second).Map(func (in interface{}) interface{} {
		return in.(int) * -1
	})

	two := createChanObs(20, time.Millisecond * 500)

	for range Merge(one, two).Subscribe().Events() {}
}
