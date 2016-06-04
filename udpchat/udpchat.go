package udpchat

const (
	ServicePort int    = 3000
	ServiceHost string = "127.0.0.1"
)

type RequestType int

const (
	ReqSendChatMsg RequestType = 1
	ReqGetHistory  RequestType = 2
)
