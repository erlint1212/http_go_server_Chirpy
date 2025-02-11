package auth

import (
    "fmt"
    "github.com/golang-jwt/jwt/v5"
    "time"
    "github.com/google/uuid"
)

type TokenType string

const (
    TokenTypeAccess TokenType = "chirpy-access"
)

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
    method := jwt.SigningMethodHS256
    claims := jwt.RegisteredClaims{
        Issuer: string(TokenTypeAccess),
        IssuedAt: jwt.NewNumericDate((time.Now().UTC())),
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()),
        Subject: userID.String(),
    }
    token := jwt.NewWithClaims(method, claims)

    signed_string, err := token.SignedString([]byte(tokenSecret))
    if err != nil {
        return "", err
    }

    return signed_string, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {

    claims := jwt.RegisteredClaims{}

    token, err := jwt.ParseWithClaims(
        tokenString, 
        &claims, 
        func(token *jwt.Token) (interface{}, error) {
            return []byte(tokenSecret), nil
    })
    if err != nil {
        return uuid.Nil, err
    }

    issuer, err := token.Claims.GetIssuer()
    if err != nil {
        return uuid.Nil, err
    }
    if issuer != string(TokenTypeAccess) {
        return uuid.Nil, fmt.Errorf("invalid issuer")
    }


    userIDString, err := token.Claims.GetSubject()
    if err != nil {
        return uuid.Nil, err
    }

    id, err := uuid.Parse(userIDString)
    if err != nil {
        return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
    }


    return id, nil
}
