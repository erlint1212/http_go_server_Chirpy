package main

import (
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
    }

    resp_user := json_user{
        ID: new_user.ID,
        CreatedAt: new_user.CreatedAt,
        UpdatedAt: new_user.UpdatedAt,
        Email: new_user.Email,
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
        ExpiresInSeconds int `json:"expires_in_seconds"`
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
    
    expires_in := time.Hour
    if params.ExpiresInSeconds != 0 {
        parsed_ExpiresInSeconds := time.Duration(params.ExpiresInSeconds) * time.Second
        expires_in = time.Duration(parsed_ExpiresInSeconds)
    }
    token, err := auth.MakeJWT(user.ID, cfg.jwt_secret, expires_in)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, fmt.Errorf("Failed to make token: %w", err))
        return
    }

    type json_user struct {
        ID        uuid.UUID `json:"id"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
        Email     string    `json:"email"`
        Token     string    `json:"token"`
    }

    resp_user := json_user{
        ID: user.ID,
        CreatedAt: user.CreatedAt,
        UpdatedAt: user.UpdatedAt,
        Email: user.Email,
        Token: token,
    }

    respondWithJSON(w, http.StatusOK, resp_user)
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

    chirps, err := cfg.db.GetAllChirps(context.Background())
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
