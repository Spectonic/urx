package urx

type FunctionOperator func(Subscriber, Notification)

func (o FunctionOperator) Notify(s Subscriber, n Notification) {
	o(s, n)
}
