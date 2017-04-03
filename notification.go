package urx

type NotificationType string
const (
	OnStart NotificationType = "start"
	OnNext NotificationType = "next"
	OnError NotificationType = "error"
	OnComplete NotificationType = "complete"
)

type Notification struct {
	Type NotificationType
	Body interface{}
}

func (n Notification) Error() error {
	return n.Body.(error)
}

func Next(body interface{}) Notification {
	return Notification{Type: OnNext, Body: body}
}

func Error(err error) Notification {
	return Notification{Type: OnError, Body: err}
}

func Complete() Notification {
	return Notification{Type: OnComplete, Body: nil}
}

func Start() Notification {
	return Notification{Type: OnStart, Body: nil}
}