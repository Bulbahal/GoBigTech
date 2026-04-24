package service

import (
	"context"
	"errors"
	"time"

	iampb "github.com/bulbahal/GoBigTech/services/iam/v1"
	"github.com/bulbahal/GoBigTech/services/iam/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IAMService описывает бизнес-логику IAM: пользователей, сессии и контакты.
// Это чистый бизнес-интерфейс без зависимостей от gRPC-прото-типов сервера.
// gRPC-слой (internal/transport/grpc) адаптирует этот интерфейс к IAMServiceServer.
type IAMService interface {
	SignIn(ctx context.Context, req *iampb.SignInRequest) (*iampb.SignInResponse, error)
	ValidateSession(ctx context.Context, req *iampb.ValidateSessionRequest) (*iampb.ValidateSessionResponse, error)
	GetUserContact(ctx context.Context, req *iampb.GetUserContactRequest) (*iampb.GetUserContactResponse, error)
}

// iamService — минимальная реализация IAMService.
// Внутри есть зависимости (PostgreSQL, Redis, логгер), которые позже будут использованы.
type iamService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
	log      *zap.Logger
}

// NewIAMService создаёт пустой каркас сервиса, готовый к расширению.
func NewIAMService(pg *pgxpool.Pool, redis *redis.Client, log *zap.Logger) IAMService {
	return &iamService{
		users:    repository.NewPostgresUserRepository(pg),
		sessions: repository.NewRedisSessionRepository(redis),
		log:      log,
	}
}

func (s *iamService) SignIn(ctx context.Context, req *iampb.SignInRequest) (*iampb.SignInResponse, error) {
	user, err := s.users.GetUserByLogin(ctx, req.GetLogin())
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Errorf(codes.Unauthenticated, "invalid login or password")
		}
		return nil, status.Errorf(codes.Internal, "get user by login: %v", err)
	}

	// Временно сравниваем пароль как есть (без хеширования).
	if req.GetPassword() != user.PasswordHash {
		return nil, status.Errorf(codes.Unauthenticated, "invalid login or password")
	}

	sessionID := uuid.NewString()

	if err := s.sessions.CreateSession(ctx, sessionID, user.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "create session: %v", err)
	}

	expiresAt := time.Now().Add(repository.SessionTTL).Unix()

	return &iampb.SignInResponse{
		SessionId:        sessionID,
		UserId:           user.ID,
		ExpiresAtUnixSec: expiresAt,
	}, nil
}

func (s *iamService) ValidateSession(ctx context.Context, req *iampb.ValidateSessionRequest) (*iampb.ValidateSessionResponse, error) {
	userID, err := s.sessions.GetUserIDBySession(ctx, req.GetSessionId())
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Сессия не найдена или истекла — считаем её невалидной без ошибки gRPC.
			return &iampb.ValidateSessionResponse{
				Valid:  false,
				UserId: "",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "get session: %v", err)
	}

	return &iampb.ValidateSessionResponse{
		Valid:  true,
		UserId: userID,
	}, nil
}

func (s *iamService) GetUserContact(ctx context.Context, req *iampb.GetUserContactRequest) (*iampb.GetUserContactResponse, error) {
	user, err := s.users.GetUserByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "get user by id: %v", err)
	}

	telegramID := ""
	if user.TelegramID != nil {
		telegramID = *user.TelegramID
	}

	return &iampb.GetUserContactResponse{
		UserId:     user.ID,
		TelegramId: telegramID,
	}, nil
}
