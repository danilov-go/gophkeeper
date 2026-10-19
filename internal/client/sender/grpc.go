package sender

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCSender struct {
	client   proto.SyncServiceClient
	conn     *grpc.ClientConn
	logger   log
	token    string
	deviceID string
	mu       sync.RWMutex
}

func NewGRPCSender(serverAddr string, l log, deviceID string) (*GRPCSender, error) {
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCSender{
		conn:     conn,
		client:   proto.NewSyncServiceClient(conn),
		logger:   l,
		deviceID: deviceID,
	}, nil
}

func (s *GRPCSender) GetSalt(ctx context.Context, login string) (string, error) {
	resp, err := s.client.GetSalt(ctx, &proto.SaltRequest{Login: login})
	if err != nil {
		return "", err
	}
	return resp.Salt, nil
}

func (s *GRPCSender) Register(ctx context.Context, login, hash, salt string) error {
	resp, err := s.client.RegisterUser(ctx, &proto.RegisterRequest{
		Login: login,
		Hash:  hash,
		Salt:  salt,
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return errors.New("логин занят")
		}
		return err
	}
	s.mu.Lock()
	s.token = strings.TrimPrefix(resp.Token, "Bearer ")
	s.mu.Unlock()
	return nil
}

func (s *GRPCSender) Auth(ctx context.Context, login, hash string) error {
	resp, err := s.client.LoginUser(ctx, &proto.LoginRequest{
		Login: login,
		Hash:  hash,
	})
	if err != nil {
		if status.Code(err) == codes.Unauthenticated {
			return errors.New("неверный логин или пароль")
		}
		return err
	}
	s.mu.Lock()
	s.token = strings.TrimPrefix(resp.Token, "Bearer ")
	s.mu.Unlock()
	return nil
}

func (s *GRPCSender) Sync(ctx context.Context, versions map[int]models.VersionTime, changelogs []models.Changelog) ([]models.CipherData, error) {
	s.mu.RLock()
	token := s.token
	s.mu.RUnlock()
	if token == "" {
		return nil, errors.New("клиент не авторизован")
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	version := make(map[int32]*proto.VersionTime)
	for k, v := range versions {
		version[int32(k)] = &proto.VersionTime{
			Version:   int32(v.Version),
			UpdatedAt: v.UpdatedAt.UnixNano(),
		}
	}
	resp, err := s.client.Pull(ctx, &proto.PullRequest{Versions: version})
	if err != nil {
		return nil, err
	}
	var cipherData []models.CipherData
	for _, sec := range resp.Secrets {
		cipherData = append(cipherData, models.CipherData{
			ID:        int(sec.Id),
			Type:      models.SecretType(sec.Type),
			Cipher:    sec.Cipher,
			UpdatedAt: time.Unix(0, sec.UpdatedAt),
			Version:   int(sec.Version),
			Deleted:   sec.Deleted,
		})
	}
	if len(changelogs) > 0 {
		var protoLogs []*proto.Changelog
		for _, log := range changelogs {
			cipherData := &proto.CipherData{
				Id:        int32(log.Secret.ID),
				Type:      string(log.Secret.Type),
				Cipher:    log.Secret.Cipher,
				UpdatedAt: log.Secret.UpdatedAt.UnixNano(),
				Version:   int32(log.Secret.Version),
				Deleted:   log.Secret.Deleted,
			}
			protoLogs = append(protoLogs, &proto.Changelog{
				Id:       int32(log.ID),
				Action:   string(log.Action),
				SecretId: int32(log.SecretID),
				Secret:   cipherData,
			})
		}
		_, err = s.client.Push(ctx, &proto.PushRequest{
			DeviceId:   s.deviceID,
			Changelogs: protoLogs,
		})
		if err != nil {
			return nil, err
		}
	}
	return cipherData, nil
}

func (s *GRPCSender) Logout() {
	s.mu.Lock()
	s.token = ""
	s.mu.Unlock()
}

func (s *GRPCSender) ServerEvent(ctx context.Context, runSync func()) {
	go func() {
		for {
			if err := ctx.Err(); err != nil {
				return
			}
			s.mu.RLock()
			token := s.token
			s.mu.RUnlock()
			if token == "" {
				time.Sleep(time.Second * 1)
				continue
			}
			md := metadata.Pairs("authorization", "Bearer "+token)
			streamCtx, streamCancel := context.WithCancel(metadata.NewOutgoingContext(ctx, md))
			stream, err := s.client.Stream(streamCtx, &proto.StreamRequest{DeviceId: s.deviceID})
			if err != nil {
				s.logger.Errorw("не удалось подключиться к серверу оповещений", "error", err)
				streamCancel()
				time.Sleep(time.Second * 5)
				continue
			}
			for {
				if err := ctx.Err(); err != nil {
					break
				}
				s.mu.RLock()
				nextToken := s.token
				s.mu.RUnlock()
				if nextToken != token {
					break
				}
				resp, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					break
				}
				if resp.Message == "sync" {
					runSync()
				}
			}
			streamCancel()
			time.Sleep(time.Second * 2)
		}
	}()
}

func (g *GRPCSender) Close() {
	if g.conn != nil {
		_ = g.conn.Close()
	}
}
