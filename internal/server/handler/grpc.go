package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	proto.UnimplementedSyncServiceServer
	service ServerService
	logger  log
}

func NewGRPCHandler(s ServerService, l log) *GRPCHandler {
	return &GRPCHandler{
		service: s,
		logger:  l,
	}
}

func (s *GRPCHandler) GetSalt(ctx context.Context, req *proto.SaltRequest) (*proto.SaltResponse, error) {
	if req.Login == "" {
		return nil, status.Error(codes.InvalidArgument, "логин не может быть пустым")
	}
	salt := s.service.GetSalt(ctx, req.Login)
	return &proto.SaltResponse{Salt: salt}, nil
}

func (s *GRPCHandler) RegisterUser(ctx context.Context, req *proto.RegisterRequest) (*proto.AuthResponse, error) {
	token, err := s.service.Register(ctx, req.Login, req.Hash, req.Salt)
	if err != nil {
		if errors.Is(err, models.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "логин занят")
		}
		s.logger.Errorw("ошибка регистрации", "error", err)
		return nil, status.Errorf(codes.Internal, "ошибка регистрации")
	}
	return &proto.AuthResponse{Token: token}, nil
}

func (s *GRPCHandler) LoginUser(ctx context.Context, req *proto.LoginRequest) (*proto.AuthResponse, error) {
	token, salt, err := s.service.Login(ctx, req.Login, req.Hash)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "неверный логин или пароль")
	}
	return &proto.AuthResponse{Token: token, Salt: salt}, nil
}

func (s *GRPCHandler) Pull(ctx context.Context, req *proto.PullRequest) (*proto.PullResponse, error) {
	userID, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	versions := make(models.SecretVersions)
	for k, v := range req.Versions {
		versions[int(k)] = int(v)
	}
	secrets, err := s.service.Pull(ctx, userID, versions)
	if err != nil {
		s.logger.Errorw("ошибка бизнес логики", "error", err)
		return nil, status.Errorf(codes.Internal, "ошибка бизнес логики")
	}
	var protoSecrets []*proto.CipherData
	for _, sec := range secrets {
		protoSecrets = append(protoSecrets, &proto.CipherData{
			Id:        int32(sec.ID),
			Type:      string(sec.Type),
			Cipher:    sec.Cipher,
			UpdatedAt: sec.UpdatedAt.UnixNano(),
			Version:   int32(sec.Version),
			Deleted:   sec.Deleted,
		})
	}
	return &proto.PullResponse{Secrets: protoSecrets}, nil
}

func (s *GRPCHandler) Push(ctx context.Context, req *proto.PushRequest) (*proto.PushResponse, error) {
	id, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	var changelogs models.Changelogs
	for _, log := range req.Changelogs {
		if log.Secret == nil {
			continue
		}
		changelogs = append(changelogs, models.Changelog{
			ID:       int(log.Id),
			Action:   models.ActionType(log.Action),
			SecretID: int(log.SecretId),
			Secret: models.CipherData{
				ID:        int(log.Secret.Id),
				Type:      models.SecretType(log.Secret.Type),
				Cipher:    log.Secret.Cipher,
				UpdatedAt: time.Unix(0, log.Secret.UpdatedAt),
				Version:   int(log.Secret.Version),
				Deleted:   log.Secret.Deleted,
			},
		})
	}
	err = s.service.Push(ctx, id, req.DeviceId, changelogs)
	if err != nil {
		s.logger.Errorw("ошибка бизнес логики", "error", err)
		return nil, status.Errorf(codes.Internal, "ошибка бизнес логики")
	}
	return &proto.PushResponse{Status: "success"}, nil
}

func (s *GRPCHandler) Stream(req *proto.StreamRequest, stream proto.SyncService_StreamServer) error {
	id, err := s.authUserID(stream.Context())
	if err != nil {
		return err
	}
	deviceID := req.DeviceId
	if deviceID == "" {
		deviceID = "unknown"
	}
	ch, unsubscribe := s.service.Event(stream.Context(), id, deviceID)
	defer unsubscribe()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			err := stream.Send(&proto.StreamResponse{
				Message: event.Message,
			})
			if err != nil {
				return err
			}
		}
	}
}

func (s *GRPCHandler) getUser(ctx context.Context) (int, error) {
	if id, ok := ctx.Value("userID").(int); ok {
		return id, nil
	}
	return 0, status.Error(codes.Unauthenticated, "пользователь не авторизован")
}

func (s *GRPCHandler) authUserID(ctx context.Context) (int, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Error(codes.Unauthenticated, "метаданные запроса отсутствуют")
	}
	auth := md.Get("authorization")
	if len(auth) == 0 || auth[0] == "" {
		return 0, status.Error(codes.Unauthenticated, "заголовок Authorization не найден")
	}
	tokenString := strings.TrimPrefix(auth[0], "Bearer ")
	authUser, err := s.service.GetUserLogin(tokenString)
	if err != nil {
		return 0, status.Error(codes.Unauthenticated, "невалидный токен")
	}
	return authUser.ID, nil
}

func AuthInterceptor(service ServerService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if strings.HasSuffix(info.FullMethod, "GetSalt") ||
			strings.HasSuffix(info.FullMethod, "RegisterUser") ||
			strings.HasSuffix(info.FullMethod, "LoginUser") {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "метаданные запроса отсутствуют")
		}
		authHeader := md.Get("authorization")
		if len(authHeader) == 0 || authHeader[0] == "" {
			return nil, status.Error(codes.Unauthenticated, "заголовок Authorization не найден")
		}
		tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
		authUser, err := service.GetUserLogin(tokenString)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "невалидный или просроченный токен")
		}
		rcCtx := context.WithValue(ctx, "userID", authUser.ID)
		return handler(rcCtx, req)
	}
}
