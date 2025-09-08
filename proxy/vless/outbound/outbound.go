package outbound

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

import (
	"bytes"
	"context"
	"encoding/base64"
	"reflect"
	"strings"
	"unsafe"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/net/packetaddr"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/retry"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	"github.com/v2fly/v2ray-core/v5/common/xudp"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/proxy"
	"github.com/v2fly/v2ray-core/v5/proxy/vless"
	"github.com/v2fly/v2ray-core/v5/proxy/vless/encoding"
	"github.com/v2fly/v2ray-core/v5/proxy/vless/encryption"
	"github.com/v2fly/v2ray-core/v5/transport"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/httpupgrade"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls/utls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/websocket"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*Config))
	}))

	common.Must(common.RegisterConfig((*SimplifiedConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		simplifiedClient := config.(*SimplifiedConfig)
		enc := simplifiedClient.Encryption
		if len(enc) == 0 {
			enc = "none"
		}
		fullClient := &Config{
			Vnext: []*protocol.ServerEndpoint{
				{
					Address: simplifiedClient.Address,
					Port:    simplifiedClient.Port,
					User: []*protocol.User{
						{
							Account: serial.ToTypedMessage(&vless.Account{
								Id:         simplifiedClient.Uuid,
								Encryption: enc,
							}),
						},
					},
				},
			},
			PacketEncoding: simplifiedClient.PacketEncoding,
		}

		return common.CreateObject(ctx, fullClient)
	}))
}

// Handler is an outbound connection handler for VLess protocol.
type Handler struct {
	serverList     *protocol.ServerList
	serverPicker   protocol.ServerPicker
	policyManager  policy.Manager
	packetEncoding packetaddr.PacketAddrType
	encryption     *encryption.ClientInstance
}

// New creates a new VLess outbound handler.
func New(ctx context.Context, config *Config) (*Handler, error) {
	serverList := protocol.NewServerList()
	for _, rec := range config.Vnext {
		s, err := protocol.NewServerSpecFromPB(rec)
		if err != nil {
			return nil, newError("failed to parse server spec").Base(err).AtError()
		}
		serverList.AddServer(s)
	}

	v := core.MustFromContext(ctx)
	handler := &Handler{
		serverList:     serverList,
		serverPicker:   protocol.NewRoundRobinServerPicker(serverList),
		policyManager:  v.GetFeature(policy.ManagerType()).(policy.Manager),
		packetEncoding: config.PacketEncoding,
	}

	for i, rec := range config.Vnext {
		for j, u := range rec.User {
			mUser, _ := u.ToMemoryUser()
			account := mUser.Account.(*vless.MemoryAccount)
			switch account.Encryption {
			case "":
				return nil, newError("empty encryption").AtError()
			case "none":
			default:
				if i > 0 || j > 0 {
					return nil, newError("encryption should have one any only one user").AtError()
				}
				encryptionStr := account.Encryption
				var xorMode, seconds uint32
				var paddingStr string
				s := strings.Split(encryptionStr, ".")
				if len(s) < 4 || s[0] != "mlkem768x25519plus" {
					return nil, newError("invalid encryption")
				}
				switch s[1] {
				case "native":
				case "xorpub":
					xorMode = 1
				case "random":
					xorMode = 2
				default:
					return nil, newError("invalid encryption")
				}
				switch s[2] {
				case "1rtt":
				case "0rtt":
					seconds = 1
				default:
					return nil, newError("invalid encryption")
				}
				padding := 0
				for _, r := range s[3:] {
					if len(r) < 20 {
						padding += len(r) + 1
						continue
					}
					if b, _ := base64.RawURLEncoding.DecodeString(r); len(b) != 32 && len(b) != 1184 {
						return nil, newError("invalid encryption")
					}
				}
				encryptionStr = encryptionStr[27+len(s[2]):]
				if padding > 0 {
					paddingStr = encryptionStr[:padding-1]
					encryptionStr = encryptionStr[padding:]
				}
				e := strings.Split(encryptionStr, ".")
				var nfsPKeysBytes [][]byte
				for _, r := range e {
					b, _ := base64.RawURLEncoding.DecodeString(r)
					nfsPKeysBytes = append(nfsPKeysBytes, b)
				}
				handler.encryption = &encryption.ClientInstance{}
				if err := handler.encryption.Init(nfsPKeysBytes, xorMode, seconds, paddingStr); err != nil {
					return nil, newError("failed to use encryption").Base(err).AtError()
				}
			}
		}
	}

	return handler, nil
}

// Process implements proxy.Outbound.Process().
func (h *Handler) Process(ctx context.Context, link *transport.Link, dialer internet.Dialer) error {
	var rec *protocol.ServerSpec
	var conn internet.Connection

	if err := retry.ExponentialBackoff(5, 200).On(func() error {
		rec = h.serverPicker.PickServer()
		var err error
		conn, err = dialer.Dial(ctx, rec.Destination())
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return newError("failed to find an available destination").Base(err).AtWarning()
	}
	defer conn.Close()

	iConn := conn
	if trackedConn, ok := iConn.(*internet.TrackedConn); ok {
		iConn = trackedConn.Conn
	}
	statConn, ok := iConn.(*internet.StatCouterConnection)
	if ok {
		iConn = statConn.Connection
	}

	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified").AtError()
	}

	target := outbound.Target
	newError("tunneling request to ", target, " via ", rec.Destination().NetAddr()).AtInfo().WriteToLog(session.ExportIDToError(ctx))

	if h.encryption != nil {
		var err error
		if conn, err = h.encryption.Handshake(conn); err != nil {
			return newError("ML-KEM-768 handshake failed").Base(err).AtInfo()
		}
	}

	command := protocol.RequestCommandTCP
	if target.Network == net.Network_UDP {
		command = protocol.RequestCommandUDP
	}
	if target.Address.Family().IsDomain() && target.Address.Domain() == "v1.mux.cool" {
		command = protocol.RequestCommandMux
	}

	request := &protocol.RequestHeader{
		Version: encoding.Version,
		User:    rec.PickUser(),
		Command: command,
		Address: target.Address,
		Port:    target.Port,
	}

	account := request.User.Account.(*vless.MemoryAccount)

	requestAddons := &encoding.Addons{
		Flow: account.Flow,
	}

	var input *bytes.Reader
	var rawInput *bytes.Buffer
	allowUDP443 := false
	switch requestAddons.Flow {
	case vless.XRV + "-udp443":
		allowUDP443 = true
		requestAddons.Flow = requestAddons.Flow[:16]
		fallthrough
	case vless.XRV:
		switch request.Command {
		case protocol.RequestCommandUDP:
			if !allowUDP443 && request.Port == 443 {
				return newError("XTLS rejected UDP/443 traffic").AtInfo()
			}
		case protocol.RequestCommandMux:
			fallthrough // let server break Mux connections that contain TCP requests
		case protocol.RequestCommandTCP:
			var t reflect.Type
			var p uintptr
			if commonConn, ok := conn.(*encryption.CommonConn); ok {
				t = reflect.TypeOf(commonConn).Elem()
				p = uintptr(unsafe.Pointer(commonConn))
			} else {
				if httpupgradeConn, ok := iConn.(*httpupgrade.Connection); ok {
					iConn = httpupgradeConn.Conn
				} else if websocketConn, ok := iConn.(*websocket.Connection); ok {
					iConn = websocketConn.Conn.NetConn()
				}
				if tlsConn, ok := iConn.(*tls.Conn); ok {
					t = reflect.TypeOf(tlsConn.Conn).Elem()
					p = uintptr(unsafe.Pointer(tlsConn.Conn))
				} else if utlsConn, ok := iConn.(utls.UTLSClientConnection); ok {
					t = reflect.TypeOf(utlsConn.Conn).Elem()
					p = uintptr(unsafe.Pointer(utlsConn.Conn))
				} else if realityConn, ok := iConn.(*reality.UConn); ok {
					t = reflect.TypeOf(realityConn.Conn).Elem()
					p = uintptr(unsafe.Pointer(realityConn.Conn))
				} else {
					return newError("XTLS only supports TLS and REALITY directly for now.").AtWarning()
				}
			}
			i, _ := t.FieldByName("input")
			r, _ := t.FieldByName("rawInput")
			input = (*bytes.Reader)(unsafe.Pointer(p + i.Offset))
			rawInput = (*bytes.Buffer)(unsafe.Pointer(p + r.Offset))
		}
	}

	sessionPolicy := h.policyManager.ForLevel(request.User.Level)
	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, sessionPolicy.Timeouts.ConnectionIdle)

	clientReader := link.Reader // .(*pipe.Reader)
	clientWriter := link.Writer // .(*pipe.Writer)

	trafficState := encoding.NewTrafficState(account.ID.Bytes())

	packetEncoding := packetaddr.PacketAddrType_None
	if command == protocol.RequestCommandUDP && request.Port > 0 {
		switch {
		case requestAddons.Flow == vless.XRV, h.packetEncoding == packetaddr.PacketAddrType_XUDP && request.Port != 53 && request.Port != 443:
			packetEncoding = h.packetEncoding
			request.Command = protocol.RequestCommandMux
			request.Address = net.DomainAddress("v1.mux.cool")
			request.Port = 0
		case h.packetEncoding == packetaddr.PacketAddrType_Packet && request.Address.Family().IsIP():
			packetEncoding = h.packetEncoding
			request.Address = net.DomainAddress(packetaddr.SeqPacketMagicAddress)
			request.Port = 0
		}
	}

	postRequest := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.DownlinkOnly)

		bufferWriter := buf.NewBufferedWriter(buf.NewWriter(conn))
		if err := encoding.EncodeRequestHeader(bufferWriter, request, requestAddons); err != nil {
			return newError("failed to encode request header").Base(err).AtWarning()
		}

		// default: serverWriter := bufferWriter
		serverWriter := encoding.EncodeBodyAddons(bufferWriter, request, requestAddons, trafficState, true, ctx, conn)
		switch packetEncoding {
		case packetaddr.PacketAddrType_Packet:
			serverWriter = packetaddr.NewPacketWriter(serverWriter, target)
		case packetaddr.PacketAddrType_XUDP:
			serverWriter = xudp.NewPacketWriter(serverWriter, target)
		}

		timeoutReader, ok := clientReader.(buf.TimeoutReader)
		if ok {
			multiBuffer, err1 := timeoutReader.ReadMultiBufferTimeout(proxy.FirstPayloadTimeout)
			if err1 == nil {
				if err := serverWriter.WriteMultiBuffer(multiBuffer); err != nil {
					return err // ...
				}
			} else if err1 != buf.ErrReadTimeout {
				return err1
			} else if requestAddons.Flow == vless.XRV {
				mb := make(buf.MultiBuffer, 1)
				newError("Insert padding with empty content to camouflage VLESS header ", mb.Len()).WriteToLog(session.ExportIDToError(ctx))
				if err := serverWriter.WriteMultiBuffer(mb); err != nil {
					return err // ...
				}
			}
		} else {
			newError("Reader is not timeout reader, will send out vless header separately from first payload").AtDebug().WriteToLog(session.ExportIDToError(ctx))
		}

		// Flush; bufferWriter.WriteMultiBuffer now is bufferWriter.writer.WriteMultiBuffer
		if err := bufferWriter.SetBuffered(false); err != nil {
			return newError("failed to write A request payload").Base(err).AtWarning()
		}

		if requestAddons.Flow == vless.XRV {
			if tlsConn, ok := iConn.(*tls.Conn); ok {
				if tlsConn.ConnectionState().Version != 0x0304 /* VersionTLS13 */ {
					return newError(`failed to use `+requestAddons.Flow+`, found outer tls version `, tlsConn.ConnectionState().Version).AtWarning()
				}
			} else if utlsConn, ok := iConn.(utls.UTLSClientConnection); ok {
				if utlsConn.ConnectionState().Version != 0x0304 /* VersionTLS13 */ {
					return newError(`failed to use `+requestAddons.Flow+`, found outer tls version `, utlsConn.ConnectionState().Version).AtWarning()
				}
			}
		}

		// from clientReader.ReadMultiBuffer to serverWriter.WriteMultiBuffer
		if err := buf.Copy(clientReader, serverWriter, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transfer request payload").Base(err).AtInfo()
		}

		return nil
	}

	getResponse := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.UplinkOnly)

		responseAddons, err := encoding.DecodeResponseHeader(conn, request)
		if err != nil {
			return newError("failed to decode response header").Base(err).AtInfo()
		}

		// default: serverReader := buf.NewReader(conn)
		serverReader := encoding.DecodeBodyAddons(conn, request, responseAddons)
		if requestAddons.Flow == vless.XRV {
			serverReader = encoding.NewVisionReader(serverReader, trafficState, false, ctx, conn, input, rawInput)
		}
		switch packetEncoding {
		case packetaddr.PacketAddrType_Packet:
			serverReader = packetaddr.NewPacketReader(serverReader)
		case packetaddr.PacketAddrType_XUDP:
			serverReader = xudp.NewPacketReader(&buf.BufferedReader{Reader: serverReader})
		}

		// from serverReader.ReadMultiBuffer to clientWriter.WriteMultiBuffer
		if err := buf.Copy(serverReader, clientWriter, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transfer response payload").Base(err).AtInfo()
		}

		return nil
	}

	if err := task.Run(ctx, postRequest, task.OnSuccess(getResponse, task.Close(clientWriter))); err != nil {
		return newError("connection ends").Base(err).AtInfo()
	}

	return nil
}
