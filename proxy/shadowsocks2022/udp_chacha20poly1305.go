package shadowsocks2022

import (
	"bytes"
	"crypto/cipher"
	"io"

	"github.com/v2fly/struc"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/crypto"
	"github.com/v2fly/v2ray-core/v5/common/net"
)

type Chacha20Poly1305UDPClientPacketProcessor struct {
	Cipher cipher.AEAD
}

func NewChacha20Poly1305UDPClientPacketProcessor(c cipher.AEAD) *Chacha20Poly1305UDPClientPacketProcessor {
	return &Chacha20Poly1305UDPClientPacketProcessor{Cipher: c}
}

type chacha20poly1305Header struct {
	SessionID     [8]byte
	PacketID      uint64
	Type          byte
	TimeStamp     uint64
	PaddingLength uint16 `struc:"sizeof=Padding"`
	Padding       []byte
}

type chacha20poly1305RespHeader struct {
	SessionID       [8]byte
	PacketID        uint64
	Type            byte
	TimeStamp       uint64
	ClientSessionID [8]byte
	PaddingLength   uint16 `struc:"sizeof=Padding"`
	Padding         []byte
}

func (p *Chacha20Poly1305UDPClientPacketProcessor) EncodeUDPRequest(request *UDPRequest, out *buf.Buffer, _ UDPClientPacketProcessorCachedStateContainer) error {
	nonce := crypto.GenerateAEADNonceWithSize(24)()
	if _, err := out.Write(nonce); err != nil {
		return newError("failed to write nonce").Base(err)
	}
	headerStruct := chacha20poly1305Header{
		SessionID:     request.SessionID,
		PacketID:      request.PacketID,
		Type:          UDPHeaderTypeClientToServerStream,
		TimeStamp:     request.TimeStamp,
		PaddingLength: 0,
		Padding:       nil,
	}
	requestBodyBuffer := buf.New()
	defer requestBodyBuffer.Release()
	if err := struc.Pack(requestBodyBuffer, &headerStruct); err != nil {
		return newError("failed to pack header").Base(err)
	}
	if err := addrParser.WriteAddressPort(requestBodyBuffer, request.Address, net.Port(request.Port)); err != nil {
		return newError("failed to write address port").Base(err)
	}
	if _, err := io.Copy(requestBodyBuffer, bytes.NewReader(request.Payload.Bytes())); err != nil {
		return newError("failed to copy payload").Base(err)
	}
	encryptedDest := out.Extend(int32(p.Cipher.Overhead()) + requestBodyBuffer.Len())
	_ = p.Cipher.Seal(encryptedDest[:0], nonce, requestBodyBuffer.Bytes(), nil)
	return nil
}

func (p *Chacha20Poly1305UDPClientPacketProcessor) DecodeUDPResp(input []byte, resp *UDPResponse, _ UDPClientPacketProcessorCachedStateContainer) error {
	decryptedDestBuffer := buf.New()
	decryptedDest := decryptedDestBuffer.Extend(int32(len(input)) - 24 - int32(p.Cipher.Overhead()))
	_, err := p.Cipher.Open(decryptedDest[:0], input[:24], input[24:], nil)
	if err != nil {
		return newError("failed to open packet").Base(err)
	}
	decryptedDestReader := bytes.NewReader(decryptedDest)
	headerStruct := chacha20poly1305RespHeader{}
	if err := struc.Unpack(decryptedDestReader, &headerStruct); err != nil {
		return newError("failed to unpack header").Base(err)
	}
	resp.TimeStamp = headerStruct.TimeStamp
	resp.SessionID = headerStruct.SessionID
	resp.PacketID = headerStruct.PacketID
	addressReaderBuf := buf.New()
	defer addressReaderBuf.Release()
	var port net.Port
	resp.Address, port, err = addrParser.ReadAddressPort(addressReaderBuf, decryptedDestReader)
	if err != nil {
		return newError("failed to read address port").Base(err)
	}
	resp.Port = int(port)
	readedLength := decryptedDestReader.Size() - int64(decryptedDestReader.Len())
	decryptedDestBuffer.Advance(int32(readedLength))
	resp.Payload = decryptedDestBuffer
	resp.ClientSessionID = headerStruct.ClientSessionID
	return nil
}
