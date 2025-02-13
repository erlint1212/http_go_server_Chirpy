package main

import (
    "database/sql"
    "github.com/erlint1212/http_go_server_Chirpy/internal/database"
    "fmt"
    "net/http"
    "log"
    "os"
    "path/filepath"
    "sync/atomic" //type that allows us to safely increment and read an integer value across multiple goroutines (HTTP requests).
    _ "github.com/lib/pq"
    "github.com/joho/godotenv"
)

type apiConfig struct {
    fileserverHits atomic.Int32
    db      *database.Queries
    jwt_secret  string
}


func check(err error) {
    if err != nil {
        log.Fatalf("error: %v\n", err)
    }
}

func main() {
    godotenv.Load()

    dbURL := os.Getenv("DB_URL")
    jwt_secret := os.Getenv("JWT_SIGNATURE")

    db, err := sql.Open("postgres", dbURL)
    check(err)

    dbQueries := database.New(db)

    const port = "8080"
    const filepathRoot= "./html/app"

    mux := http.NewServeMux()
    
    apiCfg := &apiConfig{
        fileserverHits: atomic.Int32{},
        db:                      dbQueries,
        jwt_secret:     jwt_secret,
    }
    
    handlerApp := http.StripPrefix("/app/", http.FileServer(http.Dir(filepathRoot)))

    mux.Handle("/app/", apiCfg.middlewareMetricsInc(handlerApp))

    mux.HandleFunc("GET /api/healthz", handlerHealthz)
    mux.HandleFunc("GET /admin/metrics", apiCfg.handlerHits)
    mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)
    mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)
    mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
    mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
    mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
    mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
    mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
    mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

    ex, err :=  os.Executable()
    check(err)
    filepathExec := filepath.Dir(ex)

    srv := &http.Server{
        Addr: ":8080",
        Handler: mux,
    }

    log.Printf("Serving files from %s executed from %s on port: %s\n", filepathRoot, filepathExec, port)

    err = srv.ListenAndServe()
    check(err)

    fmt.Printf("works %v\n", srv)
}
