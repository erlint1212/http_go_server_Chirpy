package auth

import (
    "crypto/rand"
    "encoding/hex"
)

func MakeRefreshToken() (string, error) {
    unencoded_key := make([]byte, 32)
    rand.Read(unencoded_key)
    hexEnc_key := hex.EncodeToString(unencoded_key)

    return hexEnc_key, nil
}
