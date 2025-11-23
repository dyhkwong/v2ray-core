package encoding

import (
	"context"
	"net/url"
	"strings"

	"google.golang.org/grpc"
)

func ServerDesc(name string) grpc.ServiceDesc {
	return grpc.ServiceDesc{
		ServiceName: name,
		HandlerType: (*GunServiceServer)(nil),
		Methods:     []grpc.MethodDesc{},
		Streams: []grpc.StreamDesc{
			{
				StreamName:    "Tun",
				Handler:       _GunService_Tun_Handler,
				ServerStreams: true,
				ClientStreams: true,
			},
		},
		Metadata: "gun.proto",
	}
}

func (c *gunServiceClient) TunCustomName(ctx context.Context, name string, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Hunk, Hunk], error) {
	stream, err := c.cc.NewStream(ctx, &ServerDesc(name).Streams[0], "/"+name+"/Tun", opts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[Hunk, Hunk]{ClientStream: stream}
	return x, nil
}

func (c *gunServiceClient) TunCustomNameX(ctx context.Context, name string, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Hunk, Hunk], error) {
	var path string
	if !strings.HasPrefix(name, "/") {
		path = "/" + url.PathEscape(name) + "/Tun"
	} else {
		path = name
	}
	stream, err := c.cc.NewStream(ctx, &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}, path, opts...)
	if err != nil {
		return nil, err
	}
	return &grpc.GenericClientStream[Hunk, Hunk]{
		ClientStream: stream,
	}, nil
}

func (c *gunMultiServiceClient) TunCustomName(ctx context.Context, name string, opts ...grpc.CallOption) (grpc.BidiStreamingClient[MultiHunk, MultiHunk], error) {
	var path string
	if !strings.HasPrefix(name, "/") {
		path = "/" + url.PathEscape(name) + "/TunMulti"
	} else {
		path = name
	}
	stream, err := c.cc.NewStream(ctx, &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}, path, opts...)
	if err != nil {
		return nil, err
	}
	return &grpc.GenericClientStream[MultiHunk, MultiHunk]{
		ClientStream: stream,
	}, nil
}

type GunServiceClientX interface {
	TunCustomName(ctx context.Context, name string, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Hunk, Hunk], error)
	Tun(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Hunk, Hunk], error)
}

type GunServiceClientXWithTunCustomNameX interface {
	GunServiceClientX
	TunCustomNameX(ctx context.Context, name string, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Hunk, Hunk], error)
}

type GunMultiServiceClientX interface {
	TunCustomName(ctx context.Context, name string, opts ...grpc.CallOption) (grpc.BidiStreamingClient[MultiHunk, MultiHunk], error)
	Tun(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[MultiHunk, MultiHunk], error)
}

func RegisterGunServiceServerX(s *grpc.Server, srv GunServiceServer, name string) {
	desc := ServerDesc(name)
	s.RegisterService(&desc, srv)
}
