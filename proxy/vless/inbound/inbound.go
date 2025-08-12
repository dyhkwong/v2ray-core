package inbound

//go:generate go run github.com/v2fly/v2ray-core/v5/common/errors/errorgen

import (
	"bytes"
	"context"
	gotls "crypto/tls"
	"encoding/base64"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/pires/go-proxyproto"
	goreality "github.com/xtls/reality"

	core "github.com/v2fly/v2ray-core/v5"
	"github.com/v2fly/v2ray-core/v5/common"
	"github.com/v2fly/v2ray-core/v5/common/buf"
	"github.com/v2fly/v2ray-core/v5/common/errors"
	"github.com/v2fly/v2ray-core/v5/common/log"
	"github.com/v2fly/v2ray-core/v5/common/net"
	"github.com/v2fly/v2ray-core/v5/common/protocol"
	"github.com/v2fly/v2ray-core/v5/common/retry"
	"github.com/v2fly/v2ray-core/v5/common/serial"
	"github.com/v2fly/v2ray-core/v5/common/session"
	"github.com/v2fly/v2ray-core/v5/common/signal"
	"github.com/v2fly/v2ray-core/v5/common/task"
	feature_inbound "github.com/v2fly/v2ray-core/v5/features/inbound"
	"github.com/v2fly/v2ray-core/v5/features/policy"
	"github.com/v2fly/v2ray-core/v5/features/routing"
	"github.com/v2fly/v2ray-core/v5/proxy/vless"
	"github.com/v2fly/v2ray-core/v5/proxy/vless/encoding"
	"github.com/v2fly/v2ray-core/v5/proxy/vless/encryption"
	"github.com/v2fly/v2ray-core/v5/transport/internet"
	"github.com/v2fly/v2ray-core/v5/transport/internet/httpupgrade"
	"github.com/v2fly/v2ray-core/v5/transport/internet/reality"
	"github.com/v2fly/v2ray-core/v5/transport/internet/tls"
	"github.com/v2fly/v2ray-core/v5/transport/internet/websocket"
)

func init() {
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*Config))
	}))

	common.Must(common.RegisterConfig((*SimplifiedConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		simplifiedServer := config.(*SimplifiedConfig)
		dec := simplifiedServer.Decryption
		if len(dec) == 0 {
			dec = "none"
		}
		fullConfig := &Config{
			Clients: func() (users []*protocol.User) {
				for _, v := range simplifiedServer.Users {
					account := &vless.Account{Id: v}
					users = append(users, &protocol.User{
						Account: serial.ToTypedMessage(account),
					})
				}
				return
			}(),
			Decryption: dec,
		}

		return common.CreateObject(ctx, fullConfig)
	}))
}

// Handler is an inbound connection handler that handles messages in VLess protocol.
type Handler struct {
	inboundHandlerManager feature_inbound.Manager
	policyManager         policy.Manager
	validator             *vless.Validator
	decryption            *encryption.ServerInstance
	fallbacks             map[string]map[string]map[string]*Fallback // or nil
	// regexps               map[string]*regexp.Regexp       // or nil
}

// New creates a new VLess inbound handler.
func New(ctx context.Context, config *Config) (*Handler, error) {
	v := core.MustFromContext(ctx)
	handler := &Handler{
		inboundHandlerManager: v.GetFeature(feature_inbound.ManagerType()).(feature_inbound.Manager),
		policyManager:         v.GetFeature(policy.ManagerType()).(policy.Manager),
		validator:             new(vless.Validator),
	}

	for _, user := range config.Clients {
		u, err := user.ToMemoryUser()
		if err != nil {
			return nil, newError("failed to get VLESS user").Base(err).AtError()
		}
		if err := handler.AddUser(ctx, u); err != nil {
			return nil, newError("failed to initiate user").Base(err).AtError()
		}
	}

	switch config.Decryption {
	case "":
		return nil, newError("empty decryption")
	case "none":
	default:
		decryptionStr := config.Decryption
		var xorMode uint32
		var secondsFrom, secondsTo int64
		var paddingStr string
		s := strings.Split(decryptionStr, ".")
		if len(s) < 4 || s[0] != "mlkem768x25519plus" {
			return nil, newError("invalid decryption")
		}
		switch s[1] {
		case "native":
		case "xorpub":
			xorMode = 1
		case "random":
			xorMode = 2
		default:
			return nil, newError("invalid decryption")
		}
		t := strings.SplitN(strings.TrimSuffix(s[2], "s"), "-", 2)
		i, err := strconv.Atoi(t[0])
		if err != nil {
			return nil, newError("invalid decryption")
		}
		secondsFrom = int64(i)
		if len(t) == 2 {
			i, err := strconv.Atoi(t[1])
			if err != nil {
				return nil, newError("invalid decryption")
			}
			secondsTo = int64(i)
		}
		padding := 0
		for _, r := range s[3:] {
			if len(r) < 20 {
				padding += len(r) + 1
				continue
			}
			if b, _ := base64.RawURLEncoding.DecodeString(r); len(b) != 32 && len(b) != 64 {
				return nil, newError("invalid decryption")
			}
		}
		decryptionStr = decryptionStr[27+len(s[2]):]
		if padding > 0 {
			paddingStr = decryptionStr[:padding-1]
			decryptionStr = decryptionStr[padding:]
		}
		var nfsSKeysBytes [][]byte
		d := strings.Split(decryptionStr, ".")
		for _, r := range d {
			b, _ := base64.RawURLEncoding.DecodeString(r)
			nfsSKeysBytes = append(nfsSKeysBytes, b)
		}
		handler.decryption = &encryption.ServerInstance{}
		if err := handler.decryption.Init(nfsSKeysBytes, xorMode, secondsFrom, secondsTo, paddingStr); err != nil {
			return nil, newError("failed to use decryption").Base(err).AtError()
		}
	}

	if config.Fallbacks != nil {
		handler.fallbacks = make(map[string]map[string]map[string]*Fallback)
		// handler.regexps = make(map[string]*regexp.Regexp)
		for _, fb := range config.Fallbacks {
			if handler.fallbacks[fb.Name] == nil {
				handler.fallbacks[fb.Name] = make(map[string]map[string]*Fallback)
			}
			if handler.fallbacks[fb.Name][fb.Alpn] == nil {
				handler.fallbacks[fb.Name][fb.Alpn] = make(map[string]*Fallback)
			}
			handler.fallbacks[fb.Name][fb.Alpn][fb.Path] = fb
			/*
				if fb.Path != "" {
					if r, err := regexp.Compile(fb.Path); err != nil {
						return nil, newError("invalid path regexp").Base(err).AtError()
					} else {
						handler.regexps[fb.Path] = r
					}
				}
			*/
		}
		if handler.fallbacks[""] != nil {
			for name, apfb := range handler.fallbacks {
				if name != "" {
					for alpn := range handler.fallbacks[""] {
						if apfb[alpn] == nil {
							apfb[alpn] = make(map[string]*Fallback)
						}
					}
				}
			}
		}
		for _, apfb := range handler.fallbacks {
			if apfb[""] != nil {
				for alpn, pfb := range apfb {
					if alpn != "" { // && alpn != "h2" {
						for path, fb := range apfb[""] {
							if pfb[path] == nil {
								pfb[path] = fb
							}
						}
					}
				}
			}
		}
		if handler.fallbacks[""] != nil {
			for name, apfb := range handler.fallbacks {
				if name != "" {
					for alpn, pfb := range handler.fallbacks[""] {
						for path, fb := range pfb {
							if apfb[alpn][path] == nil {
								apfb[alpn][path] = fb
							}
						}
					}
				}
			}
		}
	}

	return handler, nil
}

// Close implements common.Closable.Close().
func (h *Handler) Close() error {
	if h.decryption != nil {
		h.decryption.Close()
	}
	return errors.Combine(common.Close(h.validator))
}

// AddUser implements proxy.UserManager.AddUser().
func (h *Handler) AddUser(ctx context.Context, u *protocol.MemoryUser) error {
	return h.validator.Add(u)
}

// RemoveUser implements proxy.UserManager.RemoveUser().
func (h *Handler) RemoveUser(ctx context.Context, e string) error {
	return h.validator.Del(e)
}

// Network implements proxy.Inbound.Network().
func (*Handler) Network() []net.Network {
	return []net.Network{net.Network_TCP, net.Network_UNIX}
}

// Process implements proxy.Inbound.Process().
func (h *Handler) Process(ctx context.Context, network net.Network, connection internet.Connection, dispatcher routing.Dispatcher) error {
	sid := session.ExportIDToError(ctx)

	iConn := connection
	statConn, ok := iConn.(*internet.StatCouterConnection)
	if ok {
		iConn = statConn.Connection
	}

	sessionPolicy := h.policyManager.ForLevel(0)
	if err := connection.SetReadDeadline(time.Now().Add(sessionPolicy.Timeouts.Handshake)); err != nil {
		return newError("unable to set read deadline").Base(err).AtWarning()
	}

	if h.decryption != nil {
		var err error
		if connection, err = h.decryption.Handshake(connection, nil); err != nil {
			return newError("ML-KEM-768 handshake failed").Base(err).AtInfo()
		}
	}

	first := buf.FromBytes(make([]byte, buf.Size))
	first.Clear()

	firstLen, _ := first.ReadFrom(connection)
	newError("firstLen = ", firstLen).AtInfo().WriteToLog(sid)

	reader := &buf.BufferedReader{
		Reader: buf.NewReader(connection),
		Buffer: buf.MultiBuffer{first},
	}

	var request *protocol.RequestHeader
	var requestAddons *encoding.Addons
	var err error

	napfb := h.fallbacks
	isfb := napfb != nil

	if isfb && firstLen < 18 {
		err = newError("fallback directly")
	} else {
		request, requestAddons, isfb, err = encoding.DecodeRequestHeader(isfb, first, reader, h.validator)
	}

	if err != nil {
		if isfb {
			if err := connection.SetReadDeadline(time.Time{}); err != nil {
				newError("unable to set back read deadline").Base(err).AtWarning().WriteToLog(sid)
			}
			newError("fallback starts").Base(err).AtInfo().WriteToLog(sid)

			name := ""
			alpn := ""
			if tlsConn, ok := iConn.(*tls.Conn); ok {
				cs := tlsConn.ConnectionState()
				name = cs.ServerName
				alpn = cs.NegotiatedProtocol
				newError("realName = " + name).AtInfo().WriteToLog(sid)
				newError("realAlpn = " + alpn).AtInfo().WriteToLog(sid)
			} else if realityConn, ok := iConn.(*reality.Conn); ok {
				cs := realityConn.ConnectionState()
				name = cs.ServerName
				alpn = cs.NegotiatedProtocol
				newError("realName = " + name).AtInfo().WriteToLog(sid)
				newError("realAlpn = " + alpn).AtInfo().WriteToLog(sid)
			}
			name = strings.ToLower(name)
			alpn = strings.ToLower(alpn)

			if len(napfb) > 1 || napfb[""] == nil {
				if name != "" && napfb[name] == nil {
					match := ""
					for n := range napfb {
						if n != "" && strings.Contains(name, n) && len(n) > len(match) {
							match = n
						}
					}
					name = match
				}
			}

			if napfb[name] == nil {
				name = ""
			}
			apfb := napfb[name]
			if apfb == nil {
				return newError(`failed to find the default "name" config`).AtWarning()
			}

			if apfb[alpn] == nil {
				alpn = ""
			}
			pfb := apfb[alpn]
			if pfb == nil {
				return newError(`failed to find the default "alpn" config`).AtWarning()
			}

			path := ""
			if len(pfb) > 1 || pfb[""] == nil {
				/*
					if lines := bytes.Split(firstBytes, []byte{'\r', '\n'}); len(lines) > 1 {
						if s := bytes.Split(lines[0], []byte{' '}); len(s) == 3 {
							if len(s[0]) < 8 && len(s[1]) > 0 && len(s[2]) == 8 {
								newError("realPath = " + string(s[1])).AtInfo().WriteToLog(sid)
								for _, fb := range pfb {
									if fb.Path != "" && h.regexps[fb.Path].Match(s[1]) {
										path = fb.Path
										break
									}
								}
							}
						}
					}
				*/
				if firstLen >= 18 && first.Byte(4) != '*' { // not h2c
					firstBytes := first.Bytes()
					for i := 4; i <= 8; i++ { // 5 -> 9
						if firstBytes[i] == '/' && firstBytes[i-1] == ' ' {
							search := len(firstBytes)
							if search > 64 {
								search = 64 // up to about 60
							}
							for j := i + 1; j < search; j++ {
								k := firstBytes[j]
								if k == '\r' || k == '\n' { // avoid logging \r or \n
									break
								}
								if k == ' ' {
									path = string(firstBytes[i:j])
									newError("realPath = " + path).AtInfo().WriteToLog(sid)
									if pfb[path] == nil {
										path = ""
									}
									break
								}
							}
							break
						}
					}
				}
			}
			fb := pfb[path]
			if fb == nil {
				return newError(`failed to find the default "path" config`).AtWarning()
			}

			ctx, cancel := context.WithCancel(ctx)
			timer := signal.CancelAfterInactivity(ctx, cancel, sessionPolicy.Timeouts.ConnectionIdle)
			ctx = policy.ContextWithBufferPolicy(ctx, sessionPolicy.Buffer)

			var conn net.Conn
			if err := retry.ExponentialBackoff(5, 100).On(func() error {
				var dialer net.Dialer
				conn, err = dialer.DialContext(ctx, fb.Type, fb.Dest)
				if err != nil {
					return err
				}
				return nil
			}); err != nil {
				return newError("failed to dial to " + fb.Dest).Base(err).AtWarning()
			}
			defer conn.Close()

			serverReader := buf.NewReader(conn)
			serverWriter := buf.NewWriter(conn)

			postRequest := func() error {
				defer timer.SetTimeout(sessionPolicy.Timeouts.DownlinkOnly)
				if fb.Xver != 0 {
					pro := buf.New()
					defer pro.Release()

					if _, err := proxyproto.HeaderProxyFromAddrs(byte(fb.Xver), connection.RemoteAddr(), connection.LocalAddr()).WriteTo(pro); err != nil {
						return newError("failed to format PROXY protocol v", fb.Xver).Base(err).AtWarning()
					}

					if err := serverWriter.WriteMultiBuffer(buf.MultiBuffer{pro}); err != nil {
						return newError("failed to set PROXY protocol v", fb.Xver).Base(err).AtWarning()
					}
				}
				if err := buf.Copy(reader, serverWriter, buf.UpdateActivity(timer)); err != nil {
					return newError("failed to fallback request payload").Base(err).AtInfo()
				}
				return nil
			}

			writer := buf.NewWriter(connection)

			getResponse := func() error {
				defer timer.SetTimeout(sessionPolicy.Timeouts.UplinkOnly)
				if err := buf.Copy(serverReader, writer, buf.UpdateActivity(timer)); err != nil {
					return newError("failed to deliver response payload").Base(err).AtInfo()
				}
				return nil
			}

			if err := task.Run(ctx, task.OnSuccess(postRequest, task.Close(serverWriter)), task.OnSuccess(getResponse, task.Close(writer))); err != nil {
				common.Interrupt(serverReader)
				common.Interrupt(serverWriter)
				return newError("fallback ends").Base(err).AtInfo()
			}
			return nil
		}

		if errors.Cause(err) != io.EOF {
			log.Record(&log.AccessMessage{
				From:   connection.RemoteAddr(),
				To:     "",
				Status: log.AccessRejected,
				Reason: err,
			})
			err = newError("invalid request from ", connection.RemoteAddr()).Base(err).AtInfo()
		}
		return err
	}

	if err := connection.SetReadDeadline(time.Time{}); err != nil {
		newError("unable to set back read deadline").Base(err).AtWarning().WriteToLog(sid)
	}
	newError("received request for ", request.Destination()).AtInfo().WriteToLog(sid)

	inbound := session.InboundFromContext(ctx)
	if inbound == nil {
		panic("no inbound metadata")
	}
	inbound.User = request.User

	account := request.User.Account.(*vless.MemoryAccount)

	responseAddons := &encoding.Addons{}

	var input *bytes.Reader
	var rawInput *bytes.Buffer
	switch requestAddons.Flow {
	case vless.XRV:
		if account.Flow == requestAddons.Flow {
			switch request.Command {
			case protocol.RequestCommandUDP:
				return newError(requestAddons.Flow + " doesn't support UDP").AtWarning()
			case protocol.RequestCommandMux:
				fallthrough // we will break Mux connections that contain TCP requests
			case protocol.RequestCommandTCP:
				var t reflect.Type
				var p uintptr
				if commonConn, ok := connection.(*encryption.CommonConn); ok {
					t = reflect.TypeOf(commonConn).Elem()
					p = uintptr(unsafe.Pointer(commonConn))
				} else {
					if httpupgradeConn, ok := iConn.(*httpupgrade.Connection); ok {
						iConn = httpupgradeConn.Conn
					} else if websocketConn, ok := iConn.(*websocket.Connection); ok {
						iConn = websocketConn.Conn.NetConn()
					}
					if tlsConn, ok := iConn.(*tls.Conn); ok {
						if tlsConn.ConnectionState().Version != 0x0304 /* VersionTLS13 */ {
							return newError(`failed to use `+requestAddons.Flow+`, found outer tls version `, tlsConn.ConnectionState().Version).AtWarning()
						}
						t = reflect.TypeOf(tlsConn.Conn).Elem()
						p = uintptr(unsafe.Pointer(tlsConn.Conn))
					} else if realityConn, ok := iConn.(*reality.Conn); ok {
						t = reflect.TypeOf(realityConn.Conn).Elem()
						p = uintptr(unsafe.Pointer(realityConn.Conn))
					} else if gotlsConn, ok := iConn.(*gotls.Conn); ok {
						if gotlsConn.ConnectionState().Version != 0x0304 /* VersionTLS13 */ {
							return newError(`failed to use `+requestAddons.Flow+`, found outer tls version `, gotlsConn.ConnectionState().Version).AtWarning()
						}
						t = reflect.TypeOf(gotlsConn).Elem()
						p = uintptr(unsafe.Pointer(gotlsConn))
					} else if gorealityConn, ok := iConn.(*goreality.Conn); ok {
						t = reflect.TypeOf(gorealityConn).Elem()
						p = uintptr(unsafe.Pointer(gorealityConn))
					} else {
						return newError("XTLS only supports TLS and REALITY directly for now.").AtWarning()
					}
				}
				i, _ := t.FieldByName("input")
				r, _ := t.FieldByName("rawInput")
				input = (*bytes.Reader)(unsafe.Pointer(p + i.Offset))
				rawInput = (*bytes.Buffer)(unsafe.Pointer(p + r.Offset))
			}
		} else {
			return newError(account.ID.String() + " is not able to use " + requestAddons.Flow).AtWarning()
		}
	}

	if request.Command != protocol.RequestCommandMux {
		ctx = log.ContextWithAccessMessage(ctx, &log.AccessMessage{
			From:   connection.RemoteAddr(),
			To:     request.Destination(),
			Status: log.AccessAccepted,
			Reason: "",
			Email:  request.User.Email,
		})
	}

	sessionPolicy = h.policyManager.ForLevel(request.User.Level)
	ctx, cancel := context.WithCancel(ctx)
	timer := signal.CancelAfterInactivity(ctx, cancel, sessionPolicy.Timeouts.ConnectionIdle)
	ctx = policy.ContextWithBufferPolicy(ctx, sessionPolicy.Buffer)

	link, err := dispatcher.Dispatch(ctx, request.Destination())
	if err != nil {
		return newError("failed to dispatch request to ", request.Destination()).Base(err).AtWarning()
	}

	serverReader := link.Reader // .(*pipe.Reader)
	serverWriter := link.Writer // .(*pipe.Writer)

	trafficState := encoding.NewTrafficState(account.ID.Bytes())

	postRequest := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.DownlinkOnly)

		// default: clientReader := reader
		clientReader := encoding.DecodeBodyAddons(reader, request, requestAddons)

		if requestAddons.Flow == vless.XRV {
			clientReader = encoding.NewVisionReader(clientReader, trafficState, true, ctx, connection, input, rawInput)
		}

		// from clientReader.ReadMultiBuffer to serverWriter.WriteMultiBuffer
		if err := buf.Copy(clientReader, serverWriter, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transfer request payload").Base(err).AtInfo()
		}

		return nil
	}

	getResponse := func() error {
		defer timer.SetTimeout(sessionPolicy.Timeouts.UplinkOnly)

		bufferWriter := buf.NewBufferedWriter(buf.NewWriter(connection))
		if err := encoding.EncodeResponseHeader(bufferWriter, request, responseAddons); err != nil {
			return newError("failed to encode response header").Base(err).AtWarning()
		}

		// default: clientWriter := bufferWriter
		clientWriter := encoding.EncodeBodyAddons(bufferWriter, request, requestAddons, trafficState, false, ctx, connection)
		{
			multiBuffer, err := serverReader.ReadMultiBuffer()
			if err != nil {
				return err // ...
			}
			if err := clientWriter.WriteMultiBuffer(multiBuffer); err != nil {
				return err // ...
			}
		}

		// Flush; bufferWriter.WriteMultiBuffer now is bufferWriter.writer.WriteMultiBuffer
		if err := bufferWriter.SetBuffered(false); err != nil {
			return newError("failed to write A response payload").Base(err).AtWarning()
		}

		// from serverReader.ReadMultiBuffer to clientWriter.WriteMultiBuffer
		if err := buf.Copy(serverReader, clientWriter, buf.UpdateActivity(timer)); err != nil {
			return newError("failed to transfer response payload").Base(err).AtInfo()
		}

		return nil
	}

	if err := task.Run(ctx, task.OnSuccess(postRequest, task.Close(serverWriter)), getResponse); err != nil {
		common.Interrupt(serverReader)
		common.Interrupt(serverWriter)
		return newError("connection ends").Base(err).AtInfo()
	}

	return nil
}
