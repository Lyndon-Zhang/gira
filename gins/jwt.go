package gins

import (
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/Lyndon-Zhang/gira/codes"
)

type Claims struct {
	jwt.StandardClaims
	MemberId    string `json:"member_id"`
	AccountPlat string `json:"account_plat"`
	PhonePlat   int32  `json:"phone_plat"`
	DeviceId    string `json:"device_id"`
	Channel     int64  `json:"channel"`
}

// 获取远程客户端ip要设置SetTrustedProxies
// 参考 https://www.cnblogs.com/mayanan/p/15703234.html
func JWT() gin.HandlerFunc {
	return func(g *gin.Context) {
		var err error
		var claims *Claims
		token := g.GetHeader("Authorization")
		if token == "" {
			err = codes.ErrInvalidJwt
		} else {
			//TODO
			claims, err = ParseJwtToken(token, "app.Config.Jwt.Secret")
			if err != nil {
				err = codes.ErrInvalidJwt
			} else if time.Now().Unix() > claims.ExpiresAt {
				err = codes.ErrJwtExpire
			} else if claims.MemberId == "" {
				err = codes.ErrInvalidJwt
			} else {
				g.Set("MemberId", claims.MemberId)
				g.Set("Claims", claims)
			}
		}
		if err != nil {
			HttpJsonResponse(g, http.StatusUnauthorized, err, nil)
			g.Abort()
			return
		}
		g.Next()
	}
}

func GenerateJwtToken(secret string, duration time.Duration, memberId string, channel int64, accountPlat string, phonePlat int32, deviceID string) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(duration)
	claims := Claims{
		jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "gira",
		},
		memberId,
		accountPlat,
		phonePlat,
		deviceID,
		channel,
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString([]byte(secret))
	return token, err
}

func GenerateJwtRefreshToken(secret string, duration time.Duration) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(duration)
	claims := Claims{
		jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
			Issuer:    "gira",
		},
		"", // MemberId
		"", // AccountPlat
		0,  // PhonePlat
		"", // DeviceId
		0,  // Channel
	}
	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := tokenClaims.SignedString([]byte(secret))
	return token, err
}

func ParseJwtToken(token string, secret string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if tokenClaims != nil {
		if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
			return claims, nil
		}
	}
	return nil, err
}
