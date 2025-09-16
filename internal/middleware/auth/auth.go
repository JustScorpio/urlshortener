// Пакет auth содержит middleware а также вспомогательные функции для аутентификации и авторизации пользователей
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

// newJWTString - создаёт токен и возвращает его в виде строки.
func newJWTString(userID string) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// Срок окончания времени жизни токена
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenLifeTime)),
		},
		// собственное утверждение
		UserID: userID,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

// AuthMiddleware - middleware для добавления и чтения кук
//
// Deprecated: в демонтрационном варианте пользователи в БД не хранятся. Доступ к созданным урлам теряется по истечении срока токена
func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			var userID string

			needCreateCookie := false
			cookie, err := r.Cookie(jwtCookieName)
			if err != nil {
				//Нужно создать новую
				needCreateCookie = true
			} else {
				// создаём экземпляр структуры с утверждениями
				claims := &Claims{}
				// парсим из строки токена tokenString в структуру claims
				token, err := jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (interface{}, error) {
					return []byte(secretKey), nil
				})

				if err != nil || !token.Valid {
					needCreateCookie = true
				} else {
					userID = claims.UserID
				}
			}

			//Если некорректный токен - выдаём новый
			if needCreateCookie {
				userID = uuid.NewString()

				newToken, err := newJWTString(userID)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Создаем новую куку
				newCookie := &http.Cookie{
					Name:     jwtCookieName,
					Value:    newToken,
					Path:     "/",
					Expires:  time.Now().Add(tokenLifeTime), //Срок жизни куки - такой же как и у токена
					HttpOnly: true,
				}

				http.SetCookie(w, newCookie)
			}

			// Добавляем UUID в контекст запроса
			ctx := customcontext.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
