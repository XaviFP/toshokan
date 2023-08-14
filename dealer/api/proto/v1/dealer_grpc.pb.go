// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.23.0
// source: dealer.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// DealerClient is the client API for Dealer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DealerClient interface {
	Deal(ctx context.Context, in *DealRequest, opts ...grpc.CallOption) (*DealResponse, error)
	StoreAnswers(ctx context.Context, in *StoreAnswersRequest, opts ...grpc.CallOption) (*StoreAnswersResponse, error)
}

type dealerClient struct {
	cc grpc.ClientConnInterface
}

func NewDealerClient(cc grpc.ClientConnInterface) DealerClient {
	return &dealerClient{cc}
}

func (c *dealerClient) Deal(ctx context.Context, in *DealRequest, opts ...grpc.CallOption) (*DealResponse, error) {
	out := new(DealResponse)
	err := c.cc.Invoke(ctx, "/v1.Dealer/Deal", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dealerClient) StoreAnswers(ctx context.Context, in *StoreAnswersRequest, opts ...grpc.CallOption) (*StoreAnswersResponse, error) {
	out := new(StoreAnswersResponse)
	err := c.cc.Invoke(ctx, "/v1.Dealer/StoreAnswers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DealerServer is the server API for Dealer service.
// All implementations should embed UnimplementedDealerServer
// for forward compatibility
type DealerServer interface {
	Deal(context.Context, *DealRequest) (*DealResponse, error)
	StoreAnswers(context.Context, *StoreAnswersRequest) (*StoreAnswersResponse, error)
}

// UnimplementedDealerServer should be embedded to have forward compatible implementations.
type UnimplementedDealerServer struct {
}

func (UnimplementedDealerServer) Deal(context.Context, *DealRequest) (*DealResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Deal not implemented")
}
func (UnimplementedDealerServer) StoreAnswers(context.Context, *StoreAnswersRequest) (*StoreAnswersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StoreAnswers not implemented")
}

// UnsafeDealerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DealerServer will
// result in compilation errors.
type UnsafeDealerServer interface {
	mustEmbedUnimplementedDealerServer()
}

func RegisterDealerServer(s grpc.ServiceRegistrar, srv DealerServer) {
	s.RegisterService(&Dealer_ServiceDesc, srv)
}

func _Dealer_Deal_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DealRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DealerServer).Deal(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.Dealer/Deal",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DealerServer).Deal(ctx, req.(*DealRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Dealer_StoreAnswers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StoreAnswersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DealerServer).StoreAnswers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.Dealer/StoreAnswers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DealerServer).StoreAnswers(ctx, req.(*StoreAnswersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Dealer_ServiceDesc is the grpc.ServiceDesc for Dealer service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Dealer_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "v1.Dealer",
	HandlerType: (*DealerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Deal",
			Handler:    _Dealer_Deal_Handler,
		},
		{
			MethodName: "StoreAnswers",
			Handler:    _Dealer_StoreAnswers_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "dealer.proto",
}
