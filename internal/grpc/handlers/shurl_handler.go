package handler

import (
	"context"
	"errors"

	"github.com/JustScorpio/urlshortener/internal/customcontext"
	"github.com/JustScorpio/urlshortener/internal/customerrors"
	"github.com/JustScorpio/urlshortener/internal/grpc/gen"
	"github.com/JustScorpio/urlshortener/internal/models/dtos"
	"github.com/JustScorpio/urlshortener/internal/services"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ShURLHandler - обработчик входящих запросов grpc
type ShURLHandler struct {
	gen.UnimplementedURLShortenerServer
	service *services.ShURLService
	baseURL string
	// protocolPart string //gRPC не имеет части http:// или https://
}

// NewShURLHandler - инициализация хэндлера
func NewShURLHandler(service *services.ShURLService, baseURL string) *ShURLHandler {
	return &ShURLHandler{
		service: service,
		baseURL: baseURL,
	}
}

// GetFullURL - получить полный адрес
func (h *ShURLHandler) GetFullURL(ctx context.Context, req *gen.GetFullURLRequest) (*gen.GetFullURLResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	// Получение сущности из сервиса
	shURL, err := h.service.Get(ctx, req.Token)
	if err != nil {
		grpcCode := codes.Internal

		var httpErr *customerrors.HTTPError
		if errors.As(err, &httpErr) {
			grpcCode = GetGRPCStatusCode(httpErr.Code)
		}

		return nil, status.Error(grpcCode, "failed to get original URL")
	}

	return &gen.GetFullURLResponse{LongUrl: shURL.LongURL}, nil
}

// ShortenURL создает короткую ссылку
func (h *ShURLHandler) ShortenURL(ctx context.Context, req *gen.ShortenURLRequest) (*gen.ShortenURLResponse, error) {
	if req.LongUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "long_url is required")
	}

	//Создаём shurl
	userID := customcontext.GetUserID(ctx)
	shurl, err := h.service.Create(ctx, dtos.NewShURL{
		LongURL:   req.LongUrl,
		CreatedBy: userID,
	})

	//Определяем статус код
	if err != nil {
		grpcCode := codes.Internal

		var httpErr *customerrors.HTTPError
		if errors.As(err, &httpErr) {
			grpcCode = GetGRPCStatusCode(httpErr.Code)
		}

		return nil, status.Error(grpcCode, "failed shorten URL")
	}

	shortURL := h.baseURL + "/" + shurl.Token

	return &gen.ShortenURLResponse{
		ShortUrl: shortURL,
	}, nil
}

// ShortenURLsBatch создает несколько коротких ссылок
func (h *ShURLHandler) ShortenURLsBatch(ctx context.Context, req *gen.ShortenURLsBatchRequest) (*gen.ShortenURLsBatchResponse, error) {
	if len(req.Urls) == 0 {
		return nil, status.Error(codes.InvalidArgument, "urls cannot be empty")
	}

	var resBatch []*gen.BatchResponseItem

	userID := customcontext.GetUserID(ctx)

	for _, reqItem := range req.Urls {
		shurl, err := h.service.Create(ctx, dtos.NewShURL{
			LongURL:   reqItem.OriginalUrl,
			CreatedBy: userID,
		})
		if err != nil {
			grpcCode := codes.Internal

			var httpErr *customerrors.HTTPError
			if errors.As(err, &httpErr) {
				grpcCode = GetGRPCStatusCode(httpErr.Code)
			}

			return nil, status.Error(grpcCode, "failed shorten URL")
		}

		resBatch = append(resBatch, &gen.BatchResponseItem{
			CorrelationId: reqItem.CorrelationId,
			ShortUrl:      h.baseURL + "/" + shurl.Token,
		})
	}

	// Формируем ответ
	response := &gen.ShortenURLsBatchResponse{
		Urls: resBatch,
	}

	return response, nil
}

// GetShURLsByUserID возвращает ссылки пользователя
func (h *ShURLHandler) GetShURLsByUserID(ctx context.Context, req *gen.GetShURLsByUserIDRequest) (*gen.GetShURLsByUserIDResponse, error) {
	userID := customcontext.GetUserID(ctx)
	userURLs, err := h.service.GetAllShURLsByUserID(ctx, userID)
	if err != nil {
		grpcCode := codes.Internal

		var httpErr *customerrors.HTTPError
		if errors.As(err, &httpErr) {
			grpcCode = GetGRPCStatusCode(httpErr.Code)
		}

		return nil, status.Error(grpcCode, "failed to get user URLs")
	}

	// Формируем ответ
	response := &gen.GetShURLsByUserIDResponse{
		Urls: make([]*gen.UserURLItem, len(userURLs)),
	}

	for i, url := range userURLs {
		shortURL := h.baseURL + "/" + url.Token
		response.Urls[i] = &gen.UserURLItem{
			ShortUrl:    shortURL,
			OriginalUrl: url.LongURL,
		}
	}

	return response, nil
}

// GetStats возвращает статистику
func (h *ShURLHandler) GetStats(ctx context.Context, req *gen.GetStatsRequest) (*gen.GetStatsResponse, error) {
	stats, err := h.service.GetStats(ctx)
	if err != nil {
		grpcCode := codes.Internal

		var httpErr *customerrors.HTTPError
		if errors.As(err, &httpErr) {
			grpcCode = GetGRPCStatusCode(httpErr.Code)
		}

		return nil, status.Error(grpcCode, "failed to get stats")
	}

	return &gen.GetStatsResponse{
		UrlsNum:  int32(stats.URLsNum),
		UsersNum: int32(stats.UsersNum),
	}, nil
}

// DeleteMany удаляет ссылки пользователя
func (h *ShURLHandler) DeleteMany(ctx context.Context, req *gen.DeleteManyRequest) (*gen.DeleteManyResponse, error) {

	userID := customcontext.GetUserID(ctx)
	if userID == "" {
		// UserID в куке пуст
		return nil, status.Error(codes.Unauthenticated, "Unauthorized")
	}

	// Удаление сущностей
	err := h.service.Delete(ctx, req.Tokens, userID)
	if err != nil {
		grpcCode := codes.Internal

		var httpErr *customerrors.HTTPError
		if errors.As(err, &httpErr) {
			grpcCode = GetGRPCStatusCode(httpErr.Code)
		}

		return nil, status.Error(grpcCode, "Failed to delete shurl")
	}

	return &gen.DeleteManyResponse{}, nil
}

// Получить на основании http-статус кода код в системе gRPC
func GetGRPCStatusCode(httpCode int) codes.Code {
	switch httpCode {
	case 400:
		return codes.InvalidArgument
	case 201, 202:
		return codes.OK
	case 409:
		return codes.AlreadyExists
	case 410:
		return codes.FailedPrecondition
	case 500:
		return codes.Internal
	case 503:
		return codes.Unavailable
	case 401:
		return codes.Unauthenticated
	case 403:
		return codes.PermissionDenied
	default:
		return codes.Unknown
	}
}
