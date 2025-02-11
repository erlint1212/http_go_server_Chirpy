package auth

import (
    "golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
    hashed_psw, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hashed_psw), err
}
