package auth

import (
    "fmt"
    "net/http"
    "golang.org/x/crypto/bcrypt"
    "strings"
    "log"
)

func CheckPasswordHash(password, hash string) error {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err
}

func GetBearerToken(header http.Header) (string, error) {
    /*
    auths := header.Get("Authorization")
    if auths == []string{} {
        return "", fmt.Errorf("No authorization header")
    }

    token := ""
    for i := range(auths) {
        split_auths := strings.Split(auths, " ")
        if split_auths[0] == "Bearer" {
            token = split_auths[1]
            break;
        }
    }
    */
    token := header.Get("Authorization")
    if token == "" {
        return "", fmt.Errorf("No Bearer in authorization header")
    }
    
    log.Println(token)
    token = strings.Split(token, " ")[1]
    log.Println(token)
    return token, nil
}
