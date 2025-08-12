package encoding

import (
	"bytes"
	"context"
	"crypto/rand"
	gotls "crypto/tls"
	"math/big"
	"strconv"

	"github.com/pires/go-proxyproto"
	goreality "github.com/xtls/reality"

	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/features/stats"
	"github.com/v2fly/v2ray-core/v5/proxy/vless/encryption"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/httpupgrade"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/websocket"
)

var (
	Tls13SupportedVersions  = []byte{0x00, 0x2b, 0x00, 0x02, 0x03, 0x04}
	TlsClientHandShakeStart = []byte{0x16, 0x03}
	TlsServerHandShakeStart = []byte{0x16, 0x03, 0x03}
	TlsApplicationDataStart = []byte{0x17, 0x03, 0x03}

	Tls13CipherSuiteDic = map[uint16]string{
		0x1301: "TLS_AES_128_GCM_SHA256",
		0x1302: "TLS_AES_256_GCM_SHA384",
		0x1303: "TLS_CHACHA20_POLY1305_SHA256",
		0x1304: "TLS_AES_128_CCM_SHA256",
		0x1305: "TLS_AES_128_CCM_8_SHA256",
	}
)

const (
	TlsHandshakeTypeClientHello byte = 0x01
	TlsHandshakeTypeServerHello byte = 0x02

	CommandPaddingContinue byte = 0x00
	CommandPaddingEnd      byte = 0x01
	CommandPaddingDirect   byte = 0x02
)

// TrafficState is used to track uplink and downlink of one connection
// It is used by XTLS to determine if switch to raw copy mode, It is used by Vision to calculate padding
type TrafficState struct {
	UserUUID               []byte
	NumberOfPacketToFilter int
	EnableXtls             bool
	IsTLS12orAbove         bool
	IsTLS                  bool
	Cipher                 uint16
	RemainingServerHello   int32
	Inbound                InboundState
	Outbound               OutboundState
}

type InboundState struct {
	// reader link state
	WithinPaddingBuffers   bool
	UplinkReaderDirectCopy bool
	RemainingCommand       int32
	RemainingContent       int32
	RemainingPadding       int32
	CurrentCommand         int
	// write link state
	IsPadding                bool
	DownlinkWriterDirectCopy bool
}

type OutboundState struct {
	// reader link state
	WithinPaddingBuffers     bool
	DownlinkReaderDirectCopy bool
	RemainingCommand         int32
	RemainingContent         int32
	RemainingPadding         int32
	CurrentCommand           int
	// write link state
	IsPadding              bool
	UplinkWriterDirectCopy bool
}

func NewTrafficState(userUUID []byte) *TrafficState {
	return &TrafficState{
		UserUUID:               userUUID,
		NumberOfPacketToFilter: 8,
		EnableXtls:             false,
		IsTLS12orAbove:         false,
		IsTLS:                  false,
		Cipher:                 0,
		RemainingServerHello:   -1,
		Inbound: InboundState{
			WithinPaddingBuffers:     true,
			UplinkReaderDirectCopy:   false,
			RemainingCommand:         -1,
			RemainingContent:         -1,
			RemainingPadding:         -1,
			CurrentCommand:           0,
			IsPadding:                true,
			DownlinkWriterDirectCopy: false,
		},
		Outbound: OutboundState{
			WithinPaddingBuffers:     true,
			DownlinkReaderDirectCopy: false,
			RemainingCommand:         -1,
			RemainingContent:         -1,
			RemainingPadding:         -1,
			CurrentCommand:           0,
			IsPadding:                true,
			UplinkWriterDirectCopy:   false,
		},
	}
}

// VisionReader is used to read xtls vision protocol
// Note Vision probably only make sense as the inner most layer of reader, since it need assess traffic state from origin proxy traffic
type VisionReader struct {
	buf.Reader
	trafficState *TrafficState
	ctx          context.Context
	isUplink     bool
	conn         net.Conn
	input        *bytes.Reader
	rawInput     *bytes.Buffer

	// internal
	directReadCounter stats.Counter
}

func NewVisionReader(reader buf.Reader, trafficState *TrafficState, isUplink bool, ctx context.Context, conn net.Conn, input *bytes.Reader, rawInput *bytes.Buffer) *VisionReader {
	return &VisionReader{
		Reader:       reader,
		trafficState: trafficState,
		ctx:          ctx,
		isUplink:     isUplink,
		conn:         conn,
		input:        input,
		rawInput:     rawInput,
	}
}

func (w *VisionReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	buffer, err := w.Reader.ReadMultiBuffer()
	if buffer.IsEmpty() {
		return buffer, err
	}

	var withinPaddingBuffers *bool
	var remainingContent *int32
	var remainingPadding *int32
	var currentCommand *int
	var switchToDirectCopy *bool
	if w.isUplink {
		withinPaddingBuffers = &w.trafficState.Inbound.WithinPaddingBuffers
		remainingContent = &w.trafficState.Inbound.RemainingContent
		remainingPadding = &w.trafficState.Inbound.RemainingPadding
		currentCommand = &w.trafficState.Inbound.CurrentCommand
		switchToDirectCopy = &w.trafficState.Inbound.UplinkReaderDirectCopy
	} else {
		withinPaddingBuffers = &w.trafficState.Outbound.WithinPaddingBuffers
		remainingContent = &w.trafficState.Outbound.RemainingContent
		remainingPadding = &w.trafficState.Outbound.RemainingPadding
		currentCommand = &w.trafficState.Outbound.CurrentCommand
		switchToDirectCopy = &w.trafficState.Outbound.DownlinkReaderDirectCopy
	}

	if *switchToDirectCopy {
		if w.directReadCounter != nil {
			w.directReadCounter.Add(int64(buffer.Len()))
		}
		return buffer, err
	}

	if *withinPaddingBuffers || w.trafficState.NumberOfPacketToFilter > 0 {
		mb2 := make(buf.MultiBuffer, 0, len(buffer))
		for _, b := range buffer {
			newbuffer := XtlsUnpadding(b, w.trafficState, w.isUplink, w.ctx)
			if newbuffer.Len() > 0 {
				mb2 = append(mb2, newbuffer)
			}
		}
		buffer = mb2
		if *remainingContent > 0 || *remainingPadding > 0 || *currentCommand == 0 {
			*withinPaddingBuffers = true
		} else if *currentCommand == 1 {
			*withinPaddingBuffers = false
		} else if *currentCommand == 2 {
			*withinPaddingBuffers = false
			*switchToDirectCopy = true
		} else {
			newError("XtlsRead unknown command ", *currentCommand, buffer.Len()).AtInfo().WriteToLog(session.ExportIDToError(w.ctx))
		}
	}
	if w.trafficState.NumberOfPacketToFilter > 0 {
		XtlsFilterTls(buffer, w.trafficState, w.ctx)
	}

	if *switchToDirectCopy {
		// XTLS Vision processes TLS-like conn's input and rawInput
		if inputBuffer, err := buf.ReadFrom(w.input); err == nil && !inputBuffer.IsEmpty() {
			buffer, _ = buf.MergeMulti(buffer, inputBuffer)
		}
		if rawInputBuffer, err := buf.ReadFrom(w.rawInput); err == nil && !rawInputBuffer.IsEmpty() {
			buffer, _ = buf.MergeMulti(buffer, rawInputBuffer)
		}
		*w.input = bytes.Reader{} // release memory
		w.input = nil
		*w.rawInput = bytes.Buffer{} // release memory
		w.rawInput = nil

		readerConn, readCounter, _ := UnwrapRawConn(w.conn)
		w.directReadCounter = readCounter
		w.Reader = buf.NewReader(readerConn)
	}
	return buffer, err
}

// VisionWriter is used to write xtls vision protocol
// Note Vision probably only make sense as the inner most layer of writer, since it need assess traffic state from origin proxy traffic
type VisionWriter struct {
	buf.Writer
	trafficState *TrafficState
	ctx          context.Context
	isUplink     bool
	conn         net.Conn

	// internal
	writeOnceUserUUID  []byte
	directWriteCounter stats.Counter
}

func NewVisionWriter(writer buf.Writer, trafficState *TrafficState, isUplink bool, ctx context.Context, conn net.Conn) *VisionWriter {
	w := make([]byte, len(trafficState.UserUUID))
	copy(w, trafficState.UserUUID)
	return &VisionWriter{
		Writer:            writer,
		trafficState:      trafficState,
		ctx:               ctx,
		writeOnceUserUUID: w,
		isUplink:          isUplink,
		conn:              conn,
	}
}

func (w *VisionWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	var isPadding *bool
	var switchToDirectCopy *bool
	if w.isUplink {
		isPadding = &w.trafficState.Outbound.IsPadding
		switchToDirectCopy = &w.trafficState.Outbound.UplinkWriterDirectCopy
	} else {
		isPadding = &w.trafficState.Inbound.IsPadding
		switchToDirectCopy = &w.trafficState.Inbound.DownlinkWriterDirectCopy
	}

	if *switchToDirectCopy {
		rawConn, _, writerCounter := UnwrapRawConn(w.conn)
		w.Writer = buf.NewWriter(rawConn)
		w.directWriteCounter = writerCounter
		*switchToDirectCopy = false
	}
	if !mb.IsEmpty() && w.directWriteCounter != nil {
		w.directWriteCounter.Add(int64(mb.Len()))
	}

	if w.trafficState.NumberOfPacketToFilter > 0 {
		XtlsFilterTls(mb, w.trafficState, w.ctx)
	}

	if *isPadding {
		if len(mb) == 1 && mb[0] == nil {
			mb[0] = XtlsPadding(nil, CommandPaddingContinue, &w.writeOnceUserUUID, true, w.ctx) // we do a long padding to hide vless header
			return w.Writer.WriteMultiBuffer(mb)
		}
		mb = ReshapeMultiBuffer(w.ctx, mb)
		longPadding := w.trafficState.IsTLS
		for i, b := range mb {
			if w.trafficState.IsTLS && b.Len() >= 6 && bytes.Equal(TlsApplicationDataStart, b.BytesTo(3)) {
				if w.trafficState.EnableXtls {
					*switchToDirectCopy = true
				}
				var command byte = CommandPaddingContinue
				if i == len(mb)-1 {
					command = CommandPaddingEnd
					if w.trafficState.EnableXtls {
						command = CommandPaddingDirect
					}
				}
				mb[i] = XtlsPadding(b, command, &w.writeOnceUserUUID, true, w.ctx)
				*isPadding = false // padding going to end
				longPadding = false
				continue
			} else if !w.trafficState.IsTLS12orAbove && w.trafficState.NumberOfPacketToFilter <= 1 { // For compatibility with earlier vision receiver, we finish padding 1 packet early
				*isPadding = false
				mb[i] = XtlsPadding(b, CommandPaddingEnd, &w.writeOnceUserUUID, longPadding, w.ctx)
				break
			}
			var command byte = CommandPaddingContinue
			if i == len(mb)-1 && !*isPadding {
				command = CommandPaddingEnd
				if w.trafficState.EnableXtls {
					command = CommandPaddingDirect
				}
			}
			mb[i] = XtlsPadding(b, command, &w.writeOnceUserUUID, longPadding, w.ctx)
		}
	}
	return w.Writer.WriteMultiBuffer(mb)
}

// ReshapeMultiBuffer prepare multi buffer for padding stucture (max 21 bytes)
func ReshapeMultiBuffer(ctx context.Context, buffer buf.MultiBuffer) buf.MultiBuffer {
	needReshape := 0
	for _, b := range buffer {
		if b.Len() >= buf.Size-21 {
			needReshape += 1
		}
	}
	if needReshape == 0 {
		return buffer
	}
	mb2 := make(buf.MultiBuffer, 0, len(buffer)+needReshape)
	toPrint := ""
	for i, buffer1 := range buffer {
		if buffer1.Len() >= buf.Size-21 {
			index := int32(bytes.LastIndex(buffer1.Bytes(), TlsApplicationDataStart))
			if index < 21 || index > buf.Size-21 {
				index = buf.Size / 2
			}
			buffer2 := buf.New()
			buffer2.Write(buffer1.BytesFrom(index))
			buffer1.Resize(0, index)
			mb2 = append(mb2, buffer1, buffer2)
			toPrint += " " + strconv.Itoa(int(buffer1.Len())) + " " + strconv.Itoa(int(buffer2.Len()))
		} else {
			mb2 = append(mb2, buffer1)
			toPrint += " " + strconv.Itoa(int(buffer1.Len()))
		}
		buffer[i] = nil
	}
	buffer = buffer[:0]
	newError("ReshapeMultiBuffer ", toPrint).WriteToLog(session.ExportIDToError(ctx))
	return mb2
}

// XtlsPadding add padding to eliminate length siganature during tls handshake
func XtlsPadding(b *buf.Buffer, command byte, userUUID *[]byte, longPadding bool, ctx context.Context) *buf.Buffer {
	var contentLen int32 = 0
	var paddingLen int32 = 0
	if b != nil {
		contentLen = b.Len()
	}
	if contentLen < 900 && longPadding {
		l, err := rand.Int(rand.Reader, big.NewInt(500))
		if err != nil {
			newError("failed to generate padding").Base(err).WriteToLog(session.ExportIDToError(ctx))
		}
		paddingLen = int32(l.Int64()) + 900 - contentLen
	} else {
		l, err := rand.Int(rand.Reader, big.NewInt(256))
		if err != nil {
			newError("failed to generate padding").Base(err).WriteToLog(session.ExportIDToError(ctx))
		}
		paddingLen = int32(l.Int64())
	}
	if paddingLen > buf.Size-21-contentLen {
		paddingLen = buf.Size - 21 - contentLen
	}
	newbuffer := buf.New()
	if userUUID != nil {
		newbuffer.Write(*userUUID)
		*userUUID = nil
	}
	newbuffer.Write([]byte{command, byte(contentLen >> 8), byte(contentLen), byte(paddingLen >> 8), byte(paddingLen)})
	if b != nil {
		newbuffer.Write(b.Bytes())
		b.Release()
		b = nil
	}
	newbuffer.Extend(paddingLen)
	newError("XtlsPadding ", contentLen, " ", paddingLen, " ", command).WriteToLog(session.ExportIDToError(ctx))
	return newbuffer
}

// XtlsUnpadding remove padding and parse command
func XtlsUnpadding(b *buf.Buffer, s *TrafficState, isUplink bool, ctx context.Context) *buf.Buffer {
	var remainingCommand *int32
	var remainingContent *int32
	var remainingPadding *int32
	var currentCommand *int
	if isUplink {
		remainingCommand = &s.Inbound.RemainingCommand
		remainingContent = &s.Inbound.RemainingContent
		remainingPadding = &s.Inbound.RemainingPadding
		currentCommand = &s.Inbound.CurrentCommand
	} else {
		remainingCommand = &s.Outbound.RemainingCommand
		remainingContent = &s.Outbound.RemainingContent
		remainingPadding = &s.Outbound.RemainingPadding
		currentCommand = &s.Outbound.CurrentCommand
	}
	if *remainingCommand == -1 && *remainingContent == -1 && *remainingPadding == -1 { // initial state
		if b.Len() >= 21 && bytes.Equal(s.UserUUID, b.BytesTo(16)) {
			b.Advance(16)
			*remainingCommand = 5
		} else {
			return b
		}
	}
	newbuffer := buf.New()
	for b.Len() > 0 {
		if *remainingCommand > 0 {
			data, err := b.ReadByte()
			if err != nil {
				return newbuffer
			}
			switch *remainingCommand {
			case 5:
				*currentCommand = int(data)
			case 4:
				*remainingContent = int32(data) << 8
			case 3:
				*remainingContent = *remainingContent | int32(data)
			case 2:
				*remainingPadding = int32(data) << 8
			case 1:
				*remainingPadding = *remainingPadding | int32(data)
				newError("Xtls Unpadding new block, content ", *remainingContent, " padding ", *remainingPadding, " command ", *currentCommand).WriteToLog(session.ExportIDToError(ctx))
			}
			*remainingCommand--
		} else if *remainingContent > 0 {
			len := *remainingContent
			if b.Len() < len {
				len = b.Len()
			}
			data, err := b.ReadBytes(len)
			if err != nil {
				return newbuffer
			}
			newbuffer.Write(data)
			*remainingContent -= len
		} else { // remainingPadding > 0
			len := *remainingPadding
			if b.Len() < len {
				len = b.Len()
			}
			b.Advance(len)
			*remainingPadding -= len
		}
		if *remainingCommand <= 0 && *remainingContent <= 0 && *remainingPadding <= 0 { // this block done
			if *currentCommand == 0 {
				*remainingCommand = 5
			} else {
				*remainingCommand = -1 // set to initial state
				*remainingContent = -1
				*remainingPadding = -1
				if b.Len() > 0 { // shouldn't happen
					newbuffer.Write(b.Bytes())
				}
				break
			}
		}
	}
	b.Release()
	b = nil
	return newbuffer
}

// XtlsFilterTls filter and recognize tls 1.3 and other info
func XtlsFilterTls(buffer buf.MultiBuffer, trafficState *TrafficState, ctx context.Context) {
	for _, b := range buffer {
		if b == nil {
			continue
		}
		trafficState.NumberOfPacketToFilter--
		if b.Len() >= 6 {
			startsBytes := b.BytesTo(6)
			if bytes.Equal(TlsServerHandShakeStart, startsBytes[:3]) && startsBytes[5] == TlsHandshakeTypeServerHello {
				trafficState.RemainingServerHello = (int32(startsBytes[3])<<8 | int32(startsBytes[4])) + 5
				trafficState.IsTLS12orAbove = true
				trafficState.IsTLS = true
				if b.Len() >= 79 && trafficState.RemainingServerHello >= 79 {
					sessionIdLen := int32(b.Byte(43))
					cipherSuite := b.BytesRange(43+sessionIdLen+1, 43+sessionIdLen+3)
					trafficState.Cipher = uint16(cipherSuite[0])<<8 | uint16(cipherSuite[1])
				} else {
					newError("XtlsFilterTls short server hello, tls 1.2 or older? ", b.Len(), " ", trafficState.RemainingServerHello).WriteToLog(session.ExportIDToError(ctx))
				}
			} else if bytes.Equal(TlsClientHandShakeStart, startsBytes[:2]) && startsBytes[5] == TlsHandshakeTypeClientHello {
				trafficState.IsTLS = true
				newError("XtlsFilterTls found tls client hello! ", buffer.Len()).WriteToLog(session.ExportIDToError(ctx))
			}
		}
		if trafficState.RemainingServerHello > 0 {
			end := trafficState.RemainingServerHello
			if end > b.Len() {
				end = b.Len()
			}
			trafficState.RemainingServerHello -= b.Len()
			if bytes.Contains(b.BytesTo(end), Tls13SupportedVersions) {
				v, ok := Tls13CipherSuiteDic[trafficState.Cipher]
				if !ok {
					v = "Old cipher: " + strconv.FormatUint(uint64(trafficState.Cipher), 16)
				} else if v != "TLS_AES_128_CCM_8_SHA256" {
					trafficState.EnableXtls = true
				}
				newError("XtlsFilterTls found tls 1.3! ", b.Len(), " ", v).WriteToLog(session.ExportIDToError(ctx))
				trafficState.NumberOfPacketToFilter = 0
				return
			} else if trafficState.RemainingServerHello <= 0 {
				newError("XtlsFilterTls found tls 1.2! ", b.Len()).WriteToLog(session.ExportIDToError(ctx))
				trafficState.NumberOfPacketToFilter = 0
				return
			}
			newError("XtlsFilterTls inconclusive server hello ", b.Len(), " ", trafficState.RemainingServerHello).WriteToLog(session.ExportIDToError(ctx))
		}
		if trafficState.NumberOfPacketToFilter <= 0 {
			newError("XtlsFilterTls stop filtering", buffer.Len()).WriteToLog(session.ExportIDToError(ctx))
		}
	}
}

// UnwrapRawConn support unwrap stats, tls, utls, reality and proxyproto conn and get raw tcp conn from it
func UnwrapRawConn(conn net.Conn) (net.Conn, stats.Counter, stats.Counter) {
	var readCounter, writerCounter stats.Counter
	if conn != nil {
		isEncryption := false
		if commonConn, ok := conn.(*encryption.CommonConn); ok {
			conn = commonConn.Conn
			isEncryption = true
		}
		if xorConn, ok := conn.(*encryption.XorConn); ok {
			return xorConn, nil, nil // full-random xorConn should not be penetrated
		}
		statConn, ok := conn.(*internet.StatCouterConnection)
		if ok {
			conn = statConn.Connection
			readCounter = statConn.ReadCounter
			writerCounter = statConn.WriteCounter
		}
		if !isEncryption { // avoids double penetration
			if httpupgradeConn, ok := conn.(*httpupgrade.Connection); ok {
				conn = httpupgradeConn.Conn
			} else if websocketConn, ok := conn.(*websocket.Connection); ok {
				conn = websocketConn.Conn.NetConn()
			}
			if tlsConn, ok := conn.(*tls.Conn); ok {
				conn = tlsConn.NetConn()
			} else if utlsConn, ok := conn.(utls.UTLSClientConnection); ok {
				conn = utlsConn.NetConn()
			} else if realityConn, ok := conn.(*reality.Conn); ok {
				conn = realityConn.NetConn()
			} else if realityUConn, ok := conn.(*reality.UConn); ok {
				conn = realityUConn.NetConn()
			} else if gotlsConn, ok := conn.(*gotls.Conn); ok {
				conn = gotlsConn.NetConn()
			} else if gorealityConn, ok := conn.(*goreality.Conn); ok {
				conn = gorealityConn.NetConn()
			}
		}
		if pc, ok := conn.(*proxyproto.Conn); ok {
			conn = pc.Raw()
			// buf.Size > 4096, there is no need to process pc's bufReader
		}
	}
	return conn, readCounter, writerCounter
}
