package handler_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/proto"
	"github.com/danilov-go/gophkeeper/internal/server/handler"
	"github.com/danilov-go/gophkeeper/internal/server/handler/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestGRPCHandler_GetSalt(t *testing.T) {
	tests := []struct {
		name      string
		req       *proto.SaltRequest
		mockSetup func(m *mocks.ServerService)
		expCode   codes.Code
		expSalt   string
	}{
		{
			name: "положительный тест",
			req:  &proto.SaltRequest{Login: expLogin},
			mockSetup: func(m *mocks.ServerService) {
				m.On("GetSalt", mock.Anything, expLogin).Return(expSalt).Once()
			},
			expCode: codes.OK,
			expSalt: expSalt,
		},
		{
			name:      "пустой логин",
			req:       &proto.SaltRequest{Login: ""},
			mockSetup: func(m *mocks.ServerService) {},
			expCode:   codes.InvalidArgument,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewServerService(t)
			tt.mockSetup(m)
			logger := zaptest.NewLogger(t)
			h := handler.NewGRPCHandler(m, logger.Sugar())
			resp, err := h.GetSalt(context.Background(), tt.req)
			assert.Equal(t, tt.expCode, status.Code(err))
			if tt.expCode == codes.OK {
				assert.Equal(t, tt.expSalt, resp.Salt)
			}
		})
	}
}

func TestGRPCHandler_RegisterUser(t *testing.T) {
	req := &proto.RegisterRequest{Login: expLogin, Hash: expHash, Salt: expSalt}
	tests := []struct {
		name      string
		mockSetup func(m *mocks.ServerService)
		expCode   codes.Code
	}{
		{
			name: "положительный тест",
			mockSetup: func(m *mocks.ServerService) {
				m.On("Register", mock.Anything, expLogin, expHash, expSalt).Return("token", nil).Once()
			},
			expCode: codes.OK,
		},
		{
			name: "пользователь уже существует",
			mockSetup: func(m *mocks.ServerService) {
				m.On("Register", mock.Anything, expLogin, expHash, expSalt).Return("", models.ErrUserAlreadyExists).Once()
			},
			expCode: codes.AlreadyExists,
		},
		{
			name: "внутренняя ошибка сервиса",
			mockSetup: func(m *mocks.ServerService) {
				m.On("Register", mock.Anything, expLogin, expHash, expSalt).Return("", errors.New("test error")).Once()
			},
			expCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewServerService(t)
			tt.mockSetup(m)
			logger := zaptest.NewLogger(t)
			h := handler.NewGRPCHandler(m, logger.Sugar())
			_, err := h.RegisterUser(context.Background(), req)
			assert.Equal(t, tt.expCode, status.Code(err))
		})
	}
}

func TestGRPCHandler_Pull(t *testing.T) {
	req := &proto.PullRequest{Versions: map[string]*proto.VersionTime{
		secretID: {Version: 1, UpdatedAt: time.Now().UnixNano()},
	}}
	ctxUser := context.WithValue(context.Background(), "userID", userID)
	tests := []struct {
		name      string
		ctx       context.Context
		mockSetup func(m *mocks.ServerService)
		expCode   codes.Code
	}{
		{
			name: "положительный тест",
			ctx:  ctxUser,
			mockSetup: func(m *mocks.ServerService) {
				m.On("Pull", mock.Anything, userID, mock.Anything).
					Return([]models.CipherData{{ID: secretID, Version: 1, UpdatedAt: time.Now()}}, nil).Once()
			},
			expCode: codes.OK,
		},
		{
			name:      "пользователь не авторизован",
			ctx:       context.Background(),
			mockSetup: func(m *mocks.ServerService) {},
			expCode:   codes.Unauthenticated,
		},
		{
			name: "внутренняя ошибка сервиса",
			ctx:  ctxUser,
			mockSetup: func(m *mocks.ServerService) {
				m.On("Pull", mock.Anything, userID, mock.Anything).
					Return(nil, errors.New("test error")).Once()
			},
			expCode: codes.Internal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewServerService(t)
			tt.mockSetup(m)
			logger := zaptest.NewLogger(t)
			h := handler.NewGRPCHandler(m, logger.Sugar())
			_, err := h.Pull(tt.ctx, req)
			assert.Equal(t, tt.expCode, status.Code(err))
		})
	}
}

func TestGRPCHandler_Push(t *testing.T) {
	ctxUser := context.WithValue(context.Background(), "userID", userID)
	req := &proto.PushRequest{
		DeviceId: deviceID,
		Changelogs: []*proto.Changelog{
			{Id: userID, SecretId: secretID, Secret: &proto.CipherData{Id: secretID, Version: 2}},
		},
	}
	tests := []struct {
		name    string
		ctx     context.Context
		mock    func(m *mocks.ServerService)
		expCode codes.Code
	}{
		{
			name: "положительный тест",
			ctx:  ctxUser,
			mock: func(m *mocks.ServerService) {
				m.On("Push", mock.Anything, userID, deviceID, mock.Anything).Return(nil).Once()
			},
			expCode: codes.OK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewServerService(t)
			tt.mock(m)
			logger := zaptest.NewLogger(t)
			h := handler.NewGRPCHandler(m, logger.Sugar())
			_, err := h.Push(tt.ctx, req)
			assert.Equal(t, tt.expCode, status.Code(err))
		})
	}
}

func TestAuthInterceptor(t *testing.T) {
	unaryHandler := func(ctx context.Context, req any) (any, error) {
		if _, ok := req.(*proto.SaltRequest); ok {
			return "success", nil
		}
		if id, ok := ctx.Value("userID").(int); ok && id == userID {
			return "success", nil
		}
		return nil, status.Error(codes.Unauthenticated, "bad context id")
	}
	tests := []struct {
		name     string
		method   string
		req      any
		ctx      context.Context
		mock     func(m *mocks.ServerService)
		wantCode codes.Code
	}{
		{
			name:   "положительный тест",
			method: "/proto.SyncService/Pull",
			req:    &proto.PullRequest{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				"authorization": []string{token},
			}),
			mock: func(m *mocks.ServerService) {
				m.On("GetUserLogin", token).Return(models.AuthUser{ID: userID}, nil).Once()
			},
			wantCode: codes.OK,
		},
		{
			name:     "пропуск GetSalt",
			method:   "/proto.SyncService/GetSalt",
			req:      &proto.SaltRequest{Login: expLogin},
			ctx:      context.Background(),
			mock:     func(m *mocks.ServerService) {},
			wantCode: codes.OK,
		},
		{
			name:     "метаданные отсутствуют",
			method:   "/proto.SyncService/Pull",
			req:      &proto.PullRequest{},
			ctx:      context.Background(),
			mock:     func(m *mocks.ServerService) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:   "пустой заголовок авторизации",
			method: "/proto.SyncService/Pull",
			req:    &proto.PullRequest{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				"authorization": []string{""},
			}),
			mock:     func(m *mocks.ServerService) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:   "невалидный токен",
			method: "/proto.SyncService/Pull",
			req:    &proto.PullRequest{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.MD{
				"authorization": []string{token + "invalid"},
			}),
			mock: func(m *mocks.ServerService) {
				m.On("GetUserLogin", token+"invalid").Return(models.AuthUser{}, errors.New("test error")).Once()
			},
			wantCode: codes.Unauthenticated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewServerService(t)
			tt.mock(m)
			interceptor := handler.AuthInterceptor(m)
			info := &grpc.UnaryServerInfo{FullMethod: tt.method}
			_, err := interceptor(tt.ctx, tt.req, info, unaryHandler)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestGRPCHandler_Stream(t *testing.T) {
	user := models.AuthUser{ID: userID, Login: expLogin}
	serv := mocks.NewServerService(t)
	serv.On("GetUserLogin", token).
		Return(models.AuthUser{ID: user.ID, Login: user.Login}, nil).Once()
	eventChan := make(chan models.SyncEvent, 1)
	cancelCalled := false
	mockCancel := func() { cancelCalled = true }
	serv.On("Event", mock.Anything, user.ID, deviceID).
		Return((<-chan models.SyncEvent)(eventChan), mockCancel).Once()
	const bufSize = 1024 * 1024
	listen := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	h := handler.NewGRPCHandler(serv, zaptest.NewLogger(t).Sugar())
	proto.RegisterSyncServiceServer(grpcServer, h)
	go func() {
		_ = grpcServer.Serve(listen)
	}()
	t.Cleanup(func() { grpcServer.Stop() })
	conn, err := grpc.NewClient("passthrough://bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listen.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	client := proto.NewSyncServiceClient(conn)
	mdCtx := metadata.NewOutgoingContext(context.Background(), metadata.MD{
		"authorization": []string{token},
	})
	clientCtx, cancelClient := context.WithCancel(mdCtx)
	stream, err := client.Stream(clientCtx, &proto.StreamRequest{DeviceId: deviceID})
	require.NoError(t, err)
	eventChan <- models.SyncEvent{Message: "sync"}
	done := make(chan bool)
	var resp *proto.StreamResponse
	var recvErr error
	go func() {
		resp, recvErr = stream.Recv()
		done <- true
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancelClient()
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("хендлер не завершил работу после отмены контекста")
	}
	assert.NoError(t, recvErr)
	require.NotNil(t, resp)
	assert.Equal(t, "sync", resp.Message)
	assert.Eventually(t, func() bool {
		return cancelCalled
	}, 200*time.Millisecond, 5*time.Millisecond)
}
