package localdns

import (
	"golang.org/x/net/dns/dnsmessage"
)

var defaultRawQueryFunc = func(request []byte) ([]byte, error) {
	requestMsg := new(dnsmessage.Message)
	err := requestMsg.Unpack(request)
	if err != nil {
		return nil, newError("failed to parse dns request").Base(err)
	}
	newError("unsupported: need Android 10 or higher").AtError().WriteToLog()
	/*
		responseMsg := new(dnsmessage.Message)
		responseMsg.ID = requestMsg.ID
		responseMsg.RCode = dnsmessage.RCodeNotImplemented
		responseMsg.RecursionAvailable = true
		responseMsg.RecursionDesired = true
		responseMsg.Response = true
		fmt.Println(responseMsg.Pack())
	*/
	return []byte{request[0], request[1], 0x81, 0x84, 0, 0, 0, 0, 0, 0, 0, 0}, nil
}
