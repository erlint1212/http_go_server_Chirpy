package auth

import (
    "fmt"
    "net/http"
    "golang.org/x/crypto/bcrypt"
    "strings"
)

func CheckPasswordHash(password, hash string) error {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err
}

func GetBearerToken(header http.Header) (string, error) {
    auth_string := header.Get("Authorization")
    if auth_string == "" {
        return "", fmt.Errorf("No authorization header")
    }
    
    auth_string_split := strings.Split(auth_string, " ")
    if len(auth_string_split) != 2 {
        return "", fmt.Errorf("Authorization header not structured correctly, should be \"ApiKey THE_KEY_HERE\"")
    }

    if auth_string_split[0] != "Bearer" {
        return "", fmt.Errorf("No Bearer in authorization header")
    }

    token := auth_string_split[1]
    return token, nil
}

func GetAPIKey(header http.Header) (string, error) {
    auth_string := header.Get("Authorization")
    if auth_string == "" {
        return "", fmt.Errorf("No authorization header")
    }
    
    auth_string_split := strings.Split(auth_string, " ")
    if len(auth_string_split) != 2 {
        return "", fmt.Errorf("Authorization header not structured correctly, should be \"ApiKey THE_KEY_HERE\"")
    }

    if auth_string_split[0] != "ApiKey" {
        return "", fmt.Errorf("No ApiKey in authorization header")
    }

    api_key := auth_string_split[1]

    return api_key, nil
}
