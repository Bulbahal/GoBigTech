package grpc

import (
	"context"

	"github.com/bulbahal/GoBigTech/services/iam/internal/service"
	iampb "github.com/bulbahal/GoBigTech/services/iam/v1"
)

// Server адаптирует бизнес-сервис IAMService под интерфейс gRPC-генерации.
// Здесь не должно быть бизнес-логики — только делегирование в слой service.
type Server struct {
	iampb.UnimplementedIAMServiceServer

	Service service.IAMService
}

func (s *Server) SignIn(ctx context.Context, req *iampb.SignInRequest) (*iampb.SignInResponse, error) {
	return s.Service.SignIn(ctx, req)
}

func (s *Server) ValidateSession(ctx context.Context, req *iampb.ValidateSessionRequest) (*iampb.ValidateSessionResponse, error) {
	return s.Service.ValidateSession(ctx, req)
}

func (s *Server) GetUserContact(ctx context.Context, req *iampb.GetUserContactRequest) (*iampb.GetUserContactResponse, error) {
	return s.Service.GetUserContact(ctx, req)
}
