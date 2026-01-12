package grpc

import (
	"context"
	"log/slog"
	"net"

	"github.com/google/uuid"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/XaviFP/toshokan/user/api/proto/v1"
	"github.com/XaviFP/toshokan/user/internal/user"
)

type Server struct {
	GRPCAddr      string
	GRPCTransport string

	Authorizer      user.Authorizer
	Creator         user.Creator
	Repository      user.Repository
	TokenRepository user.TokenRepository

	grpcServer *grpc.Server
}

func (s *Server) Start() error {
	grpcServer := grpc.NewServer()
	pb.RegisterUserAPIServer(grpcServer, s)

	listener, err := net.Listen(s.GRPCTransport, s.GRPCAddr)
	if err != nil {
		return errors.Annotatef(err, "failed to listen on %s:%s", s.GRPCTransport, s.GRPCAddr)
	}

	defer listener.Close()

	if err := grpcServer.Serve(listener); err != nil {
		return errors.Annotatef(err, "failed to serve on %v", listener.Addr())
	}

	return nil
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

func (s *Server) GetUserID(ctx context.Context, req *pb.GetUserIDRequest) (*pb.GetUserIDResponse, error) {
	var userID uuid.UUID

	switch req.By.(type) {
	case *pb.GetUserIDRequest_Token:
		id, err := s.TokenRepository.GetUserID(ctx, req.GetToken())
		if err != nil {
			slog.Error("GetUserID: failed to get userID from token", "error", err, "stack", errors.ErrorStack(err))
			return nil, errors.Trace(err)
		}
		userID = id

	case *pb.GetUserIDRequest_Username:
		u, err := s.Repository.GetUserByUsername(ctx, req.GetUsername())
		if err != nil {
			slog.Error("GetUserID: failed to get user by username", "error", err, "username", req.GetUsername(), "stack", errors.ErrorStack(err))
			return nil, errors.Trace(err)
		}
		userID = u.ID

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid by argument")
	}

	return &pb.GetUserIDResponse{Id: userID.String()}, nil
}

func (s *Server) LogIn(ctx context.Context, req *pb.LogInRequest) (*pb.LogInResponse, error) {
	token, err := s.Authorizer.Authorize(ctx, user.AuthorizationRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		if errors.Cause(err) == user.ErrUserNotFound {
			slog.Error("LogIn: user not found", "error", err, "username", req.Username)
			return nil, status.Error(codes.NotFound, "user not found")
		}

		slog.Error("LogIn: authorization failed", "error", err, "username", req.Username, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.LogInResponse{Token: token}, nil
}

func (s *Server) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.SignUpResponse, error) {
	u, err := s.Creator.Create(ctx, user.CreateUserRequest{
		Username: req.Username,
		Password: req.Password,
		Nick:     req.Nick,
		Bio:      req.Bio,
	})

	if err != nil {
		slog.Error("SignUp: failed to create user", "error", err, "username", req.Username, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	token, err := s.TokenRepository.Generate(ctx, u.ID)
	if err != nil {
		slog.Error("SignUp: failed to generate token", "error", err, "userId", u.ID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.SignUpResponse{Token: token}, nil
}
