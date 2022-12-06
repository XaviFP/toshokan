// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: deck.proto

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

// DecksAPIClient is the client API for DecksAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DecksAPIClient interface {
	GetDeck(ctx context.Context, in *GetDeckRequest, opts ...grpc.CallOption) (*GetDeckResponse, error)
	GetDecks(ctx context.Context, in *GetDecksRequest, opts ...grpc.CallOption) (*GetDecksResponse, error)
	CreateDeck(ctx context.Context, in *CreateDeckRequest, opts ...grpc.CallOption) (*CreateDeckResponse, error)
	DeleteDeck(ctx context.Context, in *DeleteDeckRequest, opts ...grpc.CallOption) (*DeleteDeckResponse, error)
	GetPopularDecks(ctx context.Context, in *GetPopularDecksRequest, opts ...grpc.CallOption) (*GetPopularDecksResponse, error)
}

type decksAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewDecksAPIClient(cc grpc.ClientConnInterface) DecksAPIClient {
	return &decksAPIClient{cc}
}

func (c *decksAPIClient) GetDeck(ctx context.Context, in *GetDeckRequest, opts ...grpc.CallOption) (*GetDeckResponse, error) {
	out := new(GetDeckResponse)
	err := c.cc.Invoke(ctx, "/v1.DecksAPI/GetDeck", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *decksAPIClient) GetDecks(ctx context.Context, in *GetDecksRequest, opts ...grpc.CallOption) (*GetDecksResponse, error) {
	out := new(GetDecksResponse)
	err := c.cc.Invoke(ctx, "/v1.DecksAPI/GetDecks", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *decksAPIClient) CreateDeck(ctx context.Context, in *CreateDeckRequest, opts ...grpc.CallOption) (*CreateDeckResponse, error) {
	out := new(CreateDeckResponse)
	err := c.cc.Invoke(ctx, "/v1.DecksAPI/CreateDeck", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *decksAPIClient) DeleteDeck(ctx context.Context, in *DeleteDeckRequest, opts ...grpc.CallOption) (*DeleteDeckResponse, error) {
	out := new(DeleteDeckResponse)
	err := c.cc.Invoke(ctx, "/v1.DecksAPI/DeleteDeck", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *decksAPIClient) GetPopularDecks(ctx context.Context, in *GetPopularDecksRequest, opts ...grpc.CallOption) (*GetPopularDecksResponse, error) {
	out := new(GetPopularDecksResponse)
	err := c.cc.Invoke(ctx, "/v1.DecksAPI/GetPopularDecks", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DecksAPIServer is the server API for DecksAPI service.
// All implementations should embed UnimplementedDecksAPIServer
// for forward compatibility
type DecksAPIServer interface {
	GetDeck(context.Context, *GetDeckRequest) (*GetDeckResponse, error)
	GetDecks(context.Context, *GetDecksRequest) (*GetDecksResponse, error)
	CreateDeck(context.Context, *CreateDeckRequest) (*CreateDeckResponse, error)
	DeleteDeck(context.Context, *DeleteDeckRequest) (*DeleteDeckResponse, error)
	GetPopularDecks(context.Context, *GetPopularDecksRequest) (*GetPopularDecksResponse, error)
}

// UnimplementedDecksAPIServer should be embedded to have forward compatible implementations.
type UnimplementedDecksAPIServer struct {
}

func (UnimplementedDecksAPIServer) GetDeck(context.Context, *GetDeckRequest) (*GetDeckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDeck not implemented")
}
func (UnimplementedDecksAPIServer) GetDecks(context.Context, *GetDecksRequest) (*GetDecksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDecks not implemented")
}
func (UnimplementedDecksAPIServer) CreateDeck(context.Context, *CreateDeckRequest) (*CreateDeckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateDeck not implemented")
}
func (UnimplementedDecksAPIServer) DeleteDeck(context.Context, *DeleteDeckRequest) (*DeleteDeckResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteDeck not implemented")
}
func (UnimplementedDecksAPIServer) GetPopularDecks(context.Context, *GetPopularDecksRequest) (*GetPopularDecksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPopularDecks not implemented")
}

// UnsafeDecksAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DecksAPIServer will
// result in compilation errors.
type UnsafeDecksAPIServer interface {
	mustEmbedUnimplementedDecksAPIServer()
}

func RegisterDecksAPIServer(s grpc.ServiceRegistrar, srv DecksAPIServer) {
	s.RegisterService(&DecksAPI_ServiceDesc, srv)
}

func _DecksAPI_GetDeck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetDeckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DecksAPIServer).GetDeck(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.DecksAPI/GetDeck",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DecksAPIServer).GetDeck(ctx, req.(*GetDeckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DecksAPI_GetDecks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetDecksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DecksAPIServer).GetDecks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.DecksAPI/GetDecks",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DecksAPIServer).GetDecks(ctx, req.(*GetDecksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DecksAPI_CreateDeck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateDeckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DecksAPIServer).CreateDeck(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.DecksAPI/CreateDeck",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DecksAPIServer).CreateDeck(ctx, req.(*CreateDeckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DecksAPI_DeleteDeck_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteDeckRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DecksAPIServer).DeleteDeck(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.DecksAPI/DeleteDeck",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DecksAPIServer).DeleteDeck(ctx, req.(*DeleteDeckRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DecksAPI_GetPopularDecks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPopularDecksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DecksAPIServer).GetPopularDecks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.DecksAPI/GetPopularDecks",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DecksAPIServer).GetPopularDecks(ctx, req.(*GetPopularDecksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DecksAPI_ServiceDesc is the grpc.ServiceDesc for DecksAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DecksAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "v1.DecksAPI",
	HandlerType: (*DecksAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetDeck",
			Handler:    _DecksAPI_GetDeck_Handler,
		},
		{
			MethodName: "GetDecks",
			Handler:    _DecksAPI_GetDecks_Handler,
		},
		{
			MethodName: "CreateDeck",
			Handler:    _DecksAPI_CreateDeck_Handler,
		},
		{
			MethodName: "DeleteDeck",
			Handler:    _DecksAPI_DeleteDeck_Handler,
		},
		{
			MethodName: "GetPopularDecks",
			Handler:    _DecksAPI_GetPopularDecks_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "deck.proto",
}
