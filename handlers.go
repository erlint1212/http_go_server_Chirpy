package main

import (
    "sort"
    "errors"
    "database/sql"
    "time"
    "github.com/google/uuid"
    "github.com/erlint1212/http_go_server_Chirpy/internal/database"
    "github.com/erlint1212/http_go_server_Chirpy/internal/auth"
    "fmt"
    "net/http"
    "log"
    "encoding/json"
    "strings"
    "context"
)

type parameters struct {
    Body string `json:"body"`
}
type respError struct {
    Error string `json:"error"`
}

var BannedWords = [3]string{ 
    "kerfuffle",
    "sharbert",
    "fornax",
}

func respondWithError(w http.ResponseWriter, code int, err error) {
    log.Printf(err.Error())
    w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    OK, dat := marshaller(payload, w)
    if !OK {
        return
    }
    w.WriteHeader(code)
	w.Write(dat)
}

func (cfg *apiConfig) createJWTToken(w http.ResponseWriter, user_id uuid.UUID) (OK bool, token string) {
    expires_in := time.Hour
    token, err := auth.MakeJWT(user_id, cfg.jwt_secret, expires_in)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to make JWT token: %w", err))
        return false, ""
    }
    return true, token
}

func decode_json(params interface{}, w http.ResponseWriter, r *http.Request) bool {
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(params)

    if err != nil {
        err_resp := respError{
            Error: fmt.Sprintf("Error decoding parameters: %s", err),
        }
        log.Println(err_resp.Error)
        respondWithJSON(w, http.StatusInternalServerError, err_resp)
        return false
    }

    /*
    if (parameters{}) == params {
        err_resp := respError{
            Error: "All fields empty, invalid input",
        }
        respondWithJSON(w, http.StatusUnprocessableEntity, err_resp)
        return false, parameters{}
    }
    */

    return true
}

func marshaller(resp interface{}, w http.ResponseWriter) (bool, []byte) {
    dat, err := json.Marshal(resp)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, err)
        return false, nil
    }
    return true, dat
}

func validateChirp(body string) (string, error) {

    const max_len_body = 140

    if len(body) > max_len_body {
        return "", fmt.Errorf("body too long, needs to be %d or less", max_len_body)
    }

    // Extremely inneficent O(m*n), fix with tails strucutre later
    body_arr := strings.Split(body, " ")
    for i:=0; i<len(body_arr); i++ {
        for j := 0; j<len(BannedWords); j++ {
            if BannedWords[j] == strings.ToLower(body_arr[i]) {
                body_arr[i] = "****" //strings.Repeat("*", len(body_arr[i]))
                break;
            }
        }
    }
    cleaned_string := strings.Join(body_arr, " ")
    
    return cleaned_string, nil
}

func handlerHealthz(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) handlerHits(w http.ResponseWriter, req *http.Request) {
    fmt.Printf("Hits: %d\n", cfg.fileserverHits.Load())

    w.Header().Set("Content-Type", "text/html; charset=utf-8")

    p := `
    <html>
      <body>
        <h1>Welcome, Chirpy Admin</h1>
        <p>Chirpy has been visited %d times!</p>
      </body>
    </html>`

    fmt.Fprintf(w, p, cfg.fileserverHits.Load())
    //w.WriteHeader("Welcome, Chirpy Admin")
    //w.Write([]byte(fmt.Sprintf("Chirpy has been visited %d times!", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, req *http.Request) {
    _ = cfg.fileserverHits.Swap(int32(0))
    fmt.Printf("Site hits reset!\n")

    err := cfg.db.DeleteAllUsers(context.Background())
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, err)
        return 
    }

    w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func (cfg *apiConfig) handlerChangePassword(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    type parameters struct {
        Email string `json:"email"`
        Password string `json:"password"`
    }
    
    access_token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("No token found in header"))
        return
    }

    user_id, err := auth.ValidateJWT(access_token, cfg.jwt_secret)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Failed to validate JWT access token: %w", err))
        return 
    }

    params := parameters{}

    OK := decode_json(&params, w, r)
    if !OK {
        return
    }

    if params.Email == "" {
        respondWithError(w, http.StatusUnprocessableEntity, fmt.Errorf("No email set"))
        return
    }
    if params.Password == "" {
        respondWithError(w, http.StatusUnprocessableEntity, fmt.Errorf("No password set"))
        return
    }

    hashed_psw, err := auth.HashPassword(params.Password)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to hash password"))
        return
    }

    update_params := database.UpdateUserEmailPasswordByUserIDParams{
        Email:          params.Email,
        HashedPassword: hashed_psw,
        ID:             user_id,
    }

    updated_user, err := cfg.db.UpdateUserEmailPasswordByUserID(context.Background(), update_params)
    if errors.Is(err, sql.ErrNoRows) {
        respondWithError(w, http.StatusNotFound, fmt.Errorf("User of JWT token not found in database: %w", err))
        return
    }
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to update user: %s", err))
        return
    }

    type json_user struct {
        ID        uuid.UUID `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Email     string    `json:"email"`
        IsChirpyRed bool    `json:"is_chirpy_red"`
    }

    resp_user := json_user{
        ID: updated_user.ID,
        CreatedAt: updated_user.CreatedAt,
        UpdatedAt: updated_user.UpdatedAt,
        Email: updated_user.Email,
        IsChirpyRed: updated_user.IsChirpyRed,
    }

    respondWithJSON(w, http.StatusOK, resp_user)

}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")

    type parameters struct {
        Email string `json:"email"`
        Password string `json:"password"`
    }

    params := parameters{}

    OK := decode_json(&params, w, r)
    if !OK {
        return
    }

    if params.Email == "" {
        respondWithError(w, http.StatusUnprocessableEntity, fmt.Errorf("No email set"))
        return
    }
    if params.Password == "" {
        respondWithError(w, http.StatusUnprocessableEntity, fmt.Errorf("No password set"))
        return
    }

    hashed_psw, err := auth.HashPassword(params.Password)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to hash password"))
        return
    }

    user_params := database.CreateUserParams{
        ID:        uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Email:     params.Email,
        HashedPassword:  hashed_psw,
    }

    new_user, err := cfg.db.CreateUser(context.Background(), user_params)
    if err != nil {
        err_resp := respError{
            Error: fmt.Sprintf("Error creating user: %s", err),
        }
        log.Println(err_resp.Error)
        respondWithJSON(w, http.StatusInternalServerError, err_resp)
        return
    }

    type json_user struct {
        ID        uuid.UUID `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Email     string    `json:"email"`
        IsChirpyRed bool    `json:"is_chirpy_red"`
    }

    resp_user := json_user{
        ID: new_user.ID,
        CreatedAt: new_user.CreatedAt,
        UpdatedAt: new_user.UpdatedAt,
        Email: new_user.Email,
        IsChirpyRed: new_user.IsChirpyRed,
    }

    /*
    err = json.Unmarshal(new_user, &resp_user) 
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, err)
        return
    }
    */

    respondWithJSON(w, http.StatusCreated, resp_user)
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")

    type parameters struct {
        Email string `json:"email"`
        Password string `json:"password"`
    }

    params := parameters{}

    OK := decode_json(&params, w, r)
    if !OK {
        return
    }

    if params.Email == "" {
        respondWithError(w, http.StatusUnprocessableEntity, fmt.Errorf("No email set"))
        return
    }
    if params.Password == "" {
        respondWithError(w, http.StatusUnprocessableEntity, fmt.Errorf("No password set"))
        return
    }

    user, err := cfg.db.GetUserByEmail(context.Background(), params.Email)
    if errors.Is(err, sql.ErrNoRows) {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Incorrect email or password"))
        return
    }
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to retrive user: %s", err))
        return
    }

    err = auth.CheckPasswordHash(params.Password, user.HashedPassword)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Incorrect email or password"))
        return
    }

    OK, token := cfg.createJWTToken(w, user.ID)
    if !OK {
        return
    }
    
    // Never returns error, follows schema
    refresh_token_string, err := auth.MakeRefreshToken()
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create token: %w", err))
        return 
    }

    refresh_token := database.CreateRefreshTokenParams{
        Token:     refresh_token_string,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        UserID:    user.ID,
        ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
    }


    _, err = cfg.db.CreateRefreshToken(context.Background(), refresh_token)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to create refresh token in database: %w", err))
        return 
    }


    type json_user struct {
        ID        uuid.UUID `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Email     string    `json:"email"`
        Token     string    `json:"token"`
        IsChirpyRed bool    `json:"is_chirpy_red"`
        Refresh_token string `json:"refresh_token"`
    }

    resp_user := json_user{
        ID: user.ID,
        CreatedAt: user.CreatedAt,
        UpdatedAt: user.UpdatedAt,
        Email: user.Email,
        Token: token,
        IsChirpyRed: user.IsChirpyRed,
        Refresh_token: refresh_token_string,
    }

    respondWithJSON(w, http.StatusOK, resp_user)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")

    old_token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get old_token: %w", err))
        return 
    }

    old_token_full, err := cfg.db.GetRefreshTokenUserIDByToken(context.Background(), old_token)
    if errors.Is(err, sql.ErrNoRows) {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Failed to find old_token in databse: %w", err))
        return 
    }
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed while looking for old_token in database: %w", err))
        return 
    }

    if old_token_full.ExpiresAt.Before(time.Now()) {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Refresh token has expired, expired at %s", old_token_full.ExpiresAt.String()))
        return 
    }
    if old_token_full.RevokedAt.Valid {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Refresh token has been revoked, revoked at %s", old_token_full.RevokedAt.Time.String()))
        return 
    }

    OK, token := cfg.createJWTToken(w, old_token_full.UserID)
    if !OK {
        return
    }

    type json_validation_token struct {
        Token     string    `json:"token"`
    }

    resp_validation_token := json_validation_token{
        Token: token,
    }

    respondWithJSON(w, http.StatusOK, resp_validation_token)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {

    token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get token: %w", err))
        return 
    }

    _, err = cfg.db.GetRefreshTokenUserIDByToken(context.Background(), token)
    if errors.Is(err, sql.ErrNoRows) {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Failed to find token in databse: %w", err))
        return 
    }
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed while looking for old_token in database: %w", err))
        return 
    }

    token_params := database.UpdateRefreshTokenRevokedAtByTokenParams{
        Token:     token,
        RevokedAt: sql.NullTime{time.Now(), true},
    }

    err = cfg.db.UpdateRefreshTokenRevokedAtByToken(context.Background(), token_params)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to revoke refresh token in database: %w", err))
        return 
    }

    w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")

    type parameters struct {
        Body string         `json:"body"`
        UserID uuid.UUID    `json:"user_id"`
    }

    params := parameters{}
    OK := decode_json(&params, w, r)
    if !OK {
        return
    }

    token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to get token: %w", err))
        return 
    }

    id, err := auth.ValidateJWT(token, cfg.jwt_secret)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Failed to validate JWT: %w", err))
        return 
    }

    params.UserID = id

    clean_body, err := validateChirp(params.Body)
    if err != nil {
        respondWithError(w, http.StatusUnprocessableEntity, err)
        return 
    }
    params.Body = clean_body

    chirp_params := database.CreateChirpParams{
        ID:        uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Body:      params.Body,
        UserID:    params.UserID,
    }

    new_chirp, err := cfg.db.CreateChirp(context.Background(), chirp_params)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, err)
        return
    }


    type json_chirp struct {
        ID        uuid.UUID `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Body      string    `json:"body"`
        UserID    uuid.UUID `json:"user_id"`
    }

    resp_chirp := json_chirp{
        ID: new_chirp.ID,
        CreatedAt: new_chirp.CreatedAt,
        UpdatedAt: new_chirp.UpdatedAt,
        Body: new_chirp.Body,
        UserID: new_chirp.UserID,
    }

    respondWithJSON(w, http.StatusCreated, resp_chirp)
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")

    query_author_id := r.URL.Query().Get("author_id")
    query_sort := r.URL.Query().Get("sort")

	var chirps []database.Chirp

    if query_author_id == "" {
        inter_chirps, err := cfg.db.GetAllChirps(context.Background())
        if err != nil {
            respondWithError(w, http.StatusInternalServerError, err)
            return
        }
        chirps = inter_chirps
    } else {
        user_id_uuid, err := uuid.Parse(query_author_id)
        if err != nil {
            respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to parse user id from string to uuid: %s", err))
            return
        }

        inter_chirps, err := cfg.db.GetAllChirpsByUserID(context.Background(), user_id_uuid)
        if err != nil {
            respondWithError(w, http.StatusInternalServerError, err)
            return
        }
        chirps = inter_chirps
    }

    if query_sort == "desc" {
        sort.Slice(chirps, func(i, j int) bool { return chirps[i].CreatedAt.After(chirps[j].CreatedAt) })
    }

    type json_chirp struct {
        ID        uuid.UUID `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Body      string    `json:"body"`
        UserID    uuid.UUID `json:"user_id"`
    }

    log.Printf("Chirps: %d\n", len(chirps))

    out_chirps := make([]json_chirp, len(chirps))

    for i:=0; i<len(chirps); i++ {
        resp_chirp := json_chirp{
            ID: chirps[i].ID,
            CreatedAt: chirps[i].CreatedAt,
            UpdatedAt: chirps[i].UpdatedAt,
            Body: chirps[i].Body,
            UserID: chirps[i].UserID,
        }

        out_chirps[i] = resp_chirp 
    }

    respondWithJSON(w, http.StatusOK, out_chirps)
}
func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {

    chirp_id := r.PathValue("chirpID")
	chirp_id_uuid, err := uuid.Parse(chirp_id)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to parse chirp id from string to uuid: %s", err))
        return
    }

    access_token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("No token found in header"))
        return
    }

    user_id, err := auth.ValidateJWT(access_token, cfg.jwt_secret)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Failed to validate JWT access token: %w", err))
        return 
    }

    chirps, err := cfg.db.GetChirpByID(context.Background(), chirp_id_uuid)
    if errors.Is(err, sql.ErrNoRows) {
        respondWithError(w, http.StatusNotFound, fmt.Errorf("No chirp found for chirp id %v", chirp_id_uuid))
        return
    }
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, err)
        return
    }
    if chirps.UserID != user_id {
        respondWithError(w, http.StatusForbidden, fmt.Errorf("User is not owner of chirp"))
        return 
    }


    delete_arg_params := database.DeleteChirpByChirpIDAndUserIDParams{
        ID:     chirp_id_uuid,
        UserID: user_id,
    }

    err = cfg.db.DeleteChirpByChirpIDAndUserID(context.Background(), delete_arg_params)
    if errors.Is(err, sql.ErrNoRows) {
        respondWithError(w, http.StatusNotFound, fmt.Errorf("No chirp found for chirp id: %v and user id: %v", chirp_id, user_id))
        return
    }
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, err)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerUpgradeUserToRed(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")

    type parameters struct {
        Event string `json:"event"`
        Data  struct {
            UserID string `json:"user_id"`
        } `json:"data"`
    }

    params := parameters{}
    OK := decode_json(&params, w, r)
    if !OK {
        return
    }

    if params.Event != "user.upgraded" {
        respondWithError(w, http.StatusNoContent, fmt.Errorf("Only accepts user.upgrade as an event"))
        return
    }

    user_auth_key, err := auth.GetAPIKey(r.Header)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Failed to extract API key from header: %w", err))
        return
    }

    if user_auth_key != cfg.polka_key {
        respondWithError(w, http.StatusUnauthorized, fmt.Errorf("Given auth key is not valid"))
        return
    }

	user_id, err := uuid.Parse(params.Data.UserID)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to parse id from string to uuid: %s", err))
        return
    }

    arg_params := database.UpdateUserIsChirpyRedByUserIDParams{
        IsChirpyRed: true,
        ID:          user_id,
    }

    _, err = cfg.db.UpdateUserIsChirpyRedByUserID(context.Background(), arg_params)
    if errors.Is(err, sql.ErrNoRows) {
        respondWithError(w, http.StatusNotFound, fmt.Errorf("No user found for id %v", user_id))
        return
    }
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to update database: %w", err))
        return
    }

    w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {

    w.Header().Set("Content-Type", "application/json")

    idString := r.PathValue("chirpID")

	id, err := uuid.Parse(idString)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to parse id from string to uuid: %s", err))
        return
    }

    chirps, err := cfg.db.GetChirpByID(context.Background(), id)
    if errors.Is(err, sql.ErrNoRows) {
        respondWithError(w, http.StatusNotFound, fmt.Errorf("No chirp found for id %v", id))
        return
    }
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, err)
        return
    }

    type json_chirp struct {
        ID        uuid.UUID `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Body      string    `json:"body"`
        UserID    uuid.UUID `json:"user_id"`
    }

    resp_chirp := json_chirp{
        ID: chirps.ID,
        CreatedAt: chirps.CreatedAt,
        UpdatedAt: chirps.UpdatedAt,
        Body: chirps.Body,
        UserID: chirps.UserID,
    }

    respondWithJSON(w, http.StatusOK, resp_chirp)
}

