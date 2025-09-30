package encoding

import (
	"context"

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

type GunServiceClientX interface {
	TunCustomName(ctx context.Context, name string, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Hunk, Hunk], error)
	Tun(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[Hunk, Hunk], error)
}

func RegisterGunServiceServerX(s *grpc.Server, srv GunServiceServer, name string) {
	desc := ServerDesc(name)
	s.RegisterService(&desc, srv)
}
