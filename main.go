package main

import (
    "fmt"
    "net/http"
    "log"
    "os"
    "path/filepath"
    "sync/atomic" //type that allows us to safely increment and read an integer value across multiple goroutines (HTTP requests).
)

type apiConfig struct {
	fileserverHits atomic.Int32
}


func check(err error) {
    if err != nil {
        log.Fatalf("error: %v\n", err)
    }
}

func main() {

    const port = "8080"
    const filepathRoot= "./html/app"

    mux := http.NewServeMux()
    
    apiCfg := &apiConfig{
        fileserverHits: atomic.Int32{},
    }
    
    handlerApp := http.StripPrefix("/app/", http.FileServer(http.Dir(filepathRoot)))

    mux.Handle("/app/", apiCfg.middlewareMetricsInc(handlerApp))

    mux.HandleFunc("GET /api/healthz", handlerHealthz)
    mux.HandleFunc("GET /admin/metrics", apiCfg.handlerHits)
    mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
    mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

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
