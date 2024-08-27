package account

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Claims struct {
	jwt.StandardClaims
	MemberId    string `json:"member_id"`
	AccountPlat string `json:"account_plat"`
	PhonePlat   int32  `json:"phone_plat"`
	DeviceId    string `json:"device_id"`
	Channel     int64  `json:"channel"`
}

// 生成 jwt token
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

// 生成 jwt refresh token
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

// 解析 jwt token
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
