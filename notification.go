package urx

type NotificationType string
const (
	OnNext NotificationType = "next"
	OnError NotificationType = "error"
	OnComplete NotificationType = "complete"
)

type Notification struct {
	t NotificationType
	body interface{}
}

func (n Notification) Type() NotificationType {
	return n.t
}

func (n Notification) Body() interface{} {
	return n.body
}

func (n Notification) Error() error {
	return n.body.(error)
}