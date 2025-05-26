package localdns

var defaultRawQueryFunc = func(request []byte) ([]byte, error) {
	newError("unsupported: need Android 10 or higher").AtError().WriteToLog()
	/*
		responseMsg := new(dns.Msg)
		responseMsg.Id = requestMsg.Id
		responseMsg.Rcode = dns.RcodeNotImplemented
		responseMsg.RecursionAvailable = true
		responseMsg.RecursionDesired = true
		responseMsg.Response = true
		return responseMsg.Pack()
	*/
	if len(request) < 12 {
		return nil, newError("too short")
	}
	return []byte{request[0], request[1], 0x81, 0x84, 0, 0, 0, 0, 0, 0, 0, 0}, nil
}
