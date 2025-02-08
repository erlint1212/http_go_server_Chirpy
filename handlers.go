package main

import (
    "fmt"
    "net/http"
    "log"
    "encoding/json"
    "strings"
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

func marshaller(resp interface{}, w http.ResponseWriter) (bool, []byte) {
    dat, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error marshalling JSON: %s", err)
        w.WriteHeader(http.StatusInternalServerError)
        return false, nil
    }
    return true, dat
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
    log.Printf(msg)
    w.WriteHeader(code)
	w.Write([]byte(msg))
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    OK, dat := marshaller(payload, w)
    if !OK {
        return
    }
    w.WriteHeader(code)
	w.Write(dat)
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
    w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request){

    w.Header().Set("Content-Type", "application/json")

    const max_len_body = 140

    decoder := json.NewDecoder(r.Body)
    params := parameters{}
    err := decoder.Decode(&params)

    if err != nil {
        err_resp := respError{
            Error: fmt.Sprintf("Error decoding parameters: %s", err),
        }
        log.Println(err_resp.Error)
        respondWithJSON(w, http.StatusInternalServerError, err_resp)
        return
    }

    if len(params.Body) > max_len_body {
        err_resp := respError{
            Error: "Chirp is too long",
        }
        log.Println(err_resp.Error)
        respondWithJSON(w, http.StatusBadRequest, err_resp)
        return
    }

    /*
    banned_words_trie := NewTrie()
    for i := 0; i<len(BannedWords); i++ {
        banned_words_trie.Insert(BannedWords[i])
    }
    */
    body_arr := strings.Split(params.Body, " ")
    for i:=0; i<len(body_arr); i++ {
        for j := 0; j<len(BannedWords); j++ {
            if BannedWords[j] == strings.ToLower(body_arr[i]) {
                body_arr[i] = "****" //strings.Repeat("*", len(body_arr[i]))
                break;
            }
        }
    }
    cleaned_string := strings.Join(body_arr, " ")

    payload_valid := struct {
        Cleaned_body string `json:"cleaned_body"`
    }{
        Cleaned_body: cleaned_string,
    }

    respondWithJSON(w, http.StatusOK, payload_valid)
}
