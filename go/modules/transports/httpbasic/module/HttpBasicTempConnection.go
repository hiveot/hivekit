package module

// This is a temporary http connection, valid for the duration of the request
// Used to gather a response asynchronously before returning.
// // This implements the IServerConnection interface.
// type HttpBasicTempConnection struct {
// 	service.ConnectionBase
// 	Resp *msg.ResponseMessage
// }

// // Set the response to be returned with the http request
// func (c *HttpBasicTempConnection) GetResponse() *msg.ResponseMessage {
// 	c.Mux.Lock()
// 	defer c.Mux.Unlock()
// 	return c.Resp
// }

// // Send response stores the response to be returned by the request handler
// func (c *HttpBasicTempConnection) SendResponse(resp *msg.ResponseMessage) error {
// 	c.Mux.Lock()
// 	defer c.Mux.Unlock()
// 	c.Resp = resp
// 	return nil
// }

// func NewHttpBasicTempConnection() *HttpBasicTempConnection {
// 	c := &HttpBasicTempConnection{}
// 	return c
// }
