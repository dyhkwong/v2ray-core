package shadowsocks

import (
	"crypto/rand"
	"io"

	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/crypto"
	"github.com/v2fly/v2ray-core/v5/common/drain"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
)

type FullReader struct {
	reader io.Reader
	buffer []byte
}

func (r *FullReader) Read(p []byte) (n int, err error) {
	if r.buffer != nil {
		n := copy(p, r.buffer)
		if n == len(r.buffer) {
			r.buffer = nil
		} else {
			r.buffer = r.buffer[n:]
		}
		if n == len(p) {
			return n, nil
		} else {
			m, err := r.reader.Read(p[n:])
			return n + m, err
		}
	}
	return r.reader.Read(p)
}

func ReadTCPSessionMultiUser(validator *Validator, reader io.Reader) (*protocol.RequestHeader, buf.Reader, error) {
	behaviorSeed := validator.GetBehaviorSeed()

	drainer, err := drain.NewBehaviorSeedLimitedDrainer(int64(behaviorSeed), 16+38, 3266, 64)
	if err != nil {
		return nil, nil, newError("failed to initialize drainer").Base(err)
	}

	var r buf.Reader

	buffer := buf.New()
	defer buffer.Release()

	if _, err := buffer.ReadFullFrom(reader, 50); err != nil {
		drainer.AcknowledgeReceive(int(buffer.Len()))
		return nil, nil, drain.WithError(drainer, reader, err)
	}

	bs := buffer.Bytes()
	user, aead, _, ivLen, err := validator.Get(bs, protocol.RequestCommandTCP)
	if err != nil {
		drainer.AcknowledgeReceive(int(buffer.Len()))
		return nil, nil, drain.WithError(drainer, reader, err)
	}

	reader = &FullReader{reader, bs[ivLen:]}
	drainer.AcknowledgeReceive(int(ivLen))

	if aead != nil {
		auth := &crypto.AEADAuthenticator{
			AEAD:           aead,
			NonceGenerator: crypto.GenerateAEADNonceWithSize(aead.NonceSize()),
		}
		r = crypto.NewAuthenticationReader(auth, &crypto.AEADChunkSizeParser{
			Auth: auth,
		}, reader, protocol.TransferTypeStream, nil)
	} else {
		account := user.Account.(*MemoryAccount)
		iv := append([]byte(nil), buffer.BytesTo(ivLen)...)
		r, err = account.Cipher.NewDecryptionReader(account.Key, iv, reader)
		if err != nil {
			return nil, nil, drain.WithError(drainer, reader, newError("failed to initialize decoding stream").Base(err).AtError())
		}
	}

	br := &buf.BufferedReader{Reader: r}

	request := &protocol.RequestHeader{
		Version: Version,
		User:    user,
		Command: protocol.RequestCommandTCP,
	}

	drainer.AcknowledgeReceive(int(buffer.Len()))
	buffer.Clear()

	addr, port, err := addrParser.ReadAddressPort(buffer, br)
	if err != nil {
		drainer.AcknowledgeReceive(int(buffer.Len()))
		return nil, nil, drain.WithError(drainer, reader, newError("failed to read address").Base(err))
	}

	request.Address = addr
	request.Port = port

	if request.Address == nil {
		drainer.AcknowledgeReceive(int(buffer.Len()))
		return nil, nil, drain.WithError(drainer, reader, newError("invalid remote address."))
	}

	return request, br, nil
}

func WriteTCPResponseMultiUser(request *protocol.RequestHeader, writer io.Writer) (buf.Writer, error) {
	user := request.User
	account := user.Account.(*MemoryAccount)

	var iv []byte
	if account.Cipher.IVSize() > 0 {
		iv = make([]byte, account.Cipher.IVSize())
		common.Must2(rand.Read(iv))
		if ivError := account.CheckIV(iv); ivError != nil {
			return nil, newError("failed to mark outgoing iv").Base(ivError)
		}
		if err := buf.WriteAllBytes(writer, iv); err != nil {
			return nil, newError("failed to write IV.").Base(err)
		}
	}

	return account.Cipher.NewEncryptionWriter(account.Key, iv, writer)
}

func DecodeUDPPacketMultiUser(validator *Validator, payload *buf.Buffer) (*protocol.RequestHeader, *buf.Buffer, error) {
	user, _, d, _, err := validator.Get(payload.Bytes(), protocol.RequestCommandUDP)
	if err != nil {
		return nil, nil, err
	}

	account := user.Account.(*MemoryAccount)

	if account.Cipher.IsAEAD() {
		payload.Clear()
		payload.Write(d)
	} else {
		if account.Cipher.IVSize() > 0 {
			iv := make([]byte, account.Cipher.IVSize())
			copy(iv, payload.BytesTo(account.Cipher.IVSize()))
		}
		if err = account.Cipher.DecodePacket(account.Key, payload); err != nil {
			return nil, nil, newError("failed to decrypt UDP payload").Base(err)
		}
	}

	request := &protocol.RequestHeader{
		Version: Version,
		User:    user,
		Command: protocol.RequestCommandUDP,
	}

	payload.SetByte(0, payload.Byte(0)&0x0F)

	addr, port, err := addrParser.ReadAddressPort(nil, payload)
	if err != nil {
		return nil, nil, newError("failed to parse address").Base(err)
	}

	request.Address = addr
	request.Port = port

	return request, payload, nil
}
