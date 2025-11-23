package grpcserver

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"auth-service/internal/middleware"
	"auth-service/internal/models"
	authpb "auth-service/proto/auth"
)

type AuthGRPCServer struct {
	authpb.UnimplementedAuthServiceServer
	db        *sqlx.DB
	jwtSecret string
}

func NewAuthGRPCServer(db *sqlx.DB, jwtSecret string) *AuthGRPCServer {
	return &AuthGRPCServer{db: db, jwtSecret: jwtSecret}
}

func (s *AuthGRPCServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	token := req.GetToken()
	if token == "" {
		return &authpb.ValidateTokenResponse{Valid: false}, nil
	}

	claims, err := middleware.ParseToken(s.jwtSecret, token)
	if err != nil {
		return &authpb.ValidateTokenResponse{Valid: false}, nil
	}

	userID, _ := claims["user_id"].(float64)
	username, _ := claims["username"].(string)

	return &authpb.ValidateTokenResponse{Valid: true, UserId: int64(userID), Username: username}, nil
}

func (s *AuthGRPCServer) GetUser(ctx context.Context, req *authpb.GetUserRequest) (*authpb.GetUserResponse, error) {
	var user models.User
	query := `SELECT id, username, created_at FROM users WHERE id=$1`
	if err := s.db.GetContext(ctx, &user, query, req.GetUserId()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to fetch user")
	}

	return &authpb.GetUserResponse{
		Id:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}, nil
}
