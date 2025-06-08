package auth

import (
	"net/http"
	"time"

	"github.com/JustScorpio/urlshortener/internal/customcontext"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const (
	// Имя куки с JWT-токеном
	jwtCookieName = "jwt_token"
	//Время жизни токена
	tokenLifeTime = time.Hour * 3
	// Ключ для генерации и расшифровки токена (В РЕАЛЬНОМ ПРИЛОЖЕНИИ ХРАНИТЬ В НАДЁЖНОМ МЕСТЕ)
	secretKey = "supersecretkey"
)

// Claims — структура утверждений, которая включает стандартные утверждения и одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// newJWTString создаёт токен и возвращает его в виде строки.
func newJWTString(userID string) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenLifeTime)),
		},
		// собственное утверждение
		UserID: userID,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(jwtCookieName))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

// middleware для добавления и чтения кук.
// В демонтрационном варианте пользователи в БД не хранятся. Доступ к созданным урлам теряется по истечении срока токена
func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			cookie, err := r.Cookie(jwtCookieName)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// создаём экземпляр структуры с утверждениями
			claims := &Claims{}
			// парсим из строки токена tokenString в структуру claims
			token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(secretKey), nil
			})

			var userID string
			if err != nil || !token.Valid {

				userID = uuid.NewString()

				//Некорректный токен - выдаём новый
				newToken, err := newJWTString(userID)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Создаем новую куку
				newCookie := &http.Cookie{
					Name:     jwtCookieName,
					Value:    newToken,
					Expires:  time.Now().Add(tokenLifeTime),
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				}

				http.SetCookie(w, newCookie)
			} else {
				userID = claims.UserID
			}

			// Добавляем UUID в контекст запроса
			ctx := customcontext.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
