package udpchat

const (
	ServicePort int    = 3000
	ServiceHost string = "127.0.0.1"
)

type RequestType int

const (
	kReqSendChatMsg RequestType = 1
	kReqGetHistory  RequestType = 2
	kReqSendFile    RequestType = 3
	kReqSendSeg     RequestType = 4
)

type ResponseType int

const (
	kRespSendFileOK     ResponseType = 3
	kRespSendFileFailed ResponseType = 4
)
