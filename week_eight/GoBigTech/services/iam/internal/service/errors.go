package service

import "google.golang.org/grpc/codes"
import "google.golang.org/grpc/status"

// grpcUnimplemented — вспомогательная функция для возврата gRPC-ошибки Unimplemented.
// Используется в заглушках методов, пока бизнес-логика не реализована.
func grpcUnimplemented(method string) error {
	return status.Errorf(codes.Unimplemented, "%s is not implemented yet", method)
}

