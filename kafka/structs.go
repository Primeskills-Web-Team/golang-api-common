package kafka

type Event struct {
	EventName string      `json:"event_name"`
	Source    string      `json:"source"`
	Token     string      `json:"token"`
	Header    string      `json:"header"`
	Data      interface{} `json:"data"`
}
