package udpchat

// We've designed a simple file transfer protocol based on UDP. This protocol
// is far from efficiency and correctness, but it's available when client and
// server are always connected, and packets between them never lost.
//
//  Client					  			   |		Server
//  kReqSendFile <packet_id> <filename>  ---->
//				  		     			 <----  kRespSendFileOK
//				  		     			 <----  kRespSendFileFailed
//
// At this stage, client requests for permission to send file, and the
// server responses with either kRespSendFileOK that permits the request
// or kRespSendFileFailed that doesn't. We can set up an unreliable connection
// between client and server through this operation(only two-way handshake),
// TODO Three-way handshake
//
//  kReqSendSeg PACKET_ID SEG_ID SEG_CONTENT  ---->
//				  							  <----  kRespRecvSegAck PACKET_ID SEG_ID
//
// The file may firstly be segmented and each segment is identified by a unique
// PACKET_ID along with a SEG_ID that's unique within the packet, followed by the
// content of the segment. The server sends back an ack packet indicating that it has
// accepted the packet.
// When all segments are accepted, the file transfer finishes.
//
// TODO four-way handshake to terminate connection
//
//
