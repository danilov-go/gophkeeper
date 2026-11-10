package sender

import (
	"context"
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/proto"
	p "github.com/danilov-go/gophkeeper/internal/proto/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGRPCSender_Register(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(m *p.MockSyncServiceClient)
		expToken  string
		expErr    string
	}{
		{
			name: "положительный тест",
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("RegisterUser", mock.Anything, &proto.RegisterRequest{
					Login: expLogin,
					Hash:  expHash,
					Salt:  expSalt,
				}, mock.Anything).
					Return(&proto.AuthResponse{Token: "Bearer " + token}, nil)
			},
			expToken: token,
			expErr:   "",
		},
		{
			name: "логин занят",
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("RegisterUser", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.AlreadyExists, "already exists"))
			},
			expToken: "",
			expErr:   "логин занят",
		},
		{
			name: "oшибка сервера",
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("RegisterUser", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "error test"))
			},
			expToken: "",
			expErr:   "error test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &p.MockSyncServiceClient{}
			tt.mockSetup(mockClient)
			defer mockClient.AssertExpectations(t)
			logger := zaptest.NewLogger(t).Sugar()
			sender := &GRPCSender{
				client:   mockClient,
				logger:   logger,
				deviceID: deviceID,
			}
			err := sender.Register(context.Background(), expLogin, expHash, expSalt)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expToken, sender.token)
			}
		})
	}
}

func TestGRPCSender_Auth(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(m *p.MockSyncServiceClient)
		expToken  string
		expErr    string
	}{
		{
			name: "положительный тест",
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("LoginUser", mock.Anything, &proto.LoginRequest{
					Login: expLogin,
					Hash:  expHash,
				}, mock.Anything).
					Return(&proto.AuthResponse{Token: "Bearer " + token}, nil)
			},
			expToken: token,
			expErr:   "",
		},
		{
			name: "ошибка авторизации",
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("LoginUser", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Unauthenticated, "unauthenticated"))
			},
			expToken: "",
			expErr:   "неверный логин или пароль",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &p.MockSyncServiceClient{}
			tt.mockSetup(mockClient)
			defer mockClient.AssertExpectations(t)
			logger := zaptest.NewLogger(t).Sugar()
			sender := &GRPCSender{
				client:   mockClient,
				logger:   logger,
				deviceID: deviceID,
			}
			err := sender.Auth(context.Background(), expLogin, expHash)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expToken, sender.token)
			}
		})
	}
}

func TestGRPCSender_GetSalt(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(m *p.MockSyncServiceClient)
		expSalt   string
		expErr    string
	}{
		{
			name: "положительный тест",
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("GetSalt", mock.Anything, &proto.SaltRequest{Login: expLogin}, mock.Anything).
					Return(&proto.SaltResponse{Salt: expSalt}, nil)
			},
			expSalt: expSalt,
			expErr:  "",
		},
		{
			name: "Ошибка сервера",
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("GetSalt", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "error test"))
			},
			expSalt: "",
			expErr:  "error test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &p.MockSyncServiceClient{}
			tt.mockSetup(mockClient)
			defer mockClient.AssertExpectations(t)
			logger := zaptest.NewLogger(t).Sugar()
			sender := &GRPCSender{
				client:   mockClient,
				logger:   logger,
				deviceID: deviceID,
			}
			salt, err := sender.GetSalt(context.Background(), expLogin)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expSalt, salt)
			}
		})
	}
}

func TestGRPCSender_Sync(t *testing.T) {
	tests := []struct {
		name      string
		expToken  string
		mockSetup func(m *p.MockSyncServiceClient)
		expLen    int
		expErr    string
	}{
		{
			name:     "положительный тест",
			expToken: token,
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("Pull", mock.MatchedBy(func(ctx context.Context) bool {
					md, ok := metadata.FromOutgoingContext(ctx)
					return ok && len(md.Get("authorization")) > 0 && md.Get("authorization")[0] == "Bearer "+token
				}), mock.MatchedBy(func(req *proto.PullRequest) bool {
					_, ok := req.Versions[secretID]
					return ok
				}), mock.Anything).
					Return(&proto.PullResponse{
						Secrets: []*proto.CipherData{
							{Id: secretID, Type: string(models.TypeCard), Version: 1},
							{Id: secretID + "1", Type: string(models.TypeBinary), Version: 1},
						},
					}, nil)
			},
			expLen: 2,
			expErr: "",
		},
		{
			name:     "ошибка сервера",
			expToken: token,
			mockSetup: func(m *p.MockSyncServiceClient) {
				m.On("Pull", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, status.Error(codes.Internal, "error test"))
			},
			expLen: 0,
			expErr: "error test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &p.MockSyncServiceClient{}
			tt.mockSetup(mockClient)
			defer mockClient.AssertExpectations(t)
			logger := zaptest.NewLogger(t).Sugar()
			sender := &GRPCSender{
				client:   mockClient,
				logger:   logger,
				deviceID: deviceID,
				token:    tt.expToken,
			}
			versions := map[string]models.VersionTime{secretID: {Version: 1, UpdatedAt: time.Now()}}
			cipherData, err := sender.Sync(context.Background(), versions, nil)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				assert.NoError(t, err)
				assert.Len(t, cipherData, tt.expLen)
			}
		})
	}
}

func TestGRPCSender_Logout(t *testing.T) {
	sender := &GRPCSender{token: token}
	sender.Logout()
	assert.Empty(t, sender.token)
}
