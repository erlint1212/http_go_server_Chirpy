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

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
        fmt.Printf("Site visited! Hits: %d\n", cfg.fileserverHits.Load())
		next.ServeHTTP(w, r)
	})
}

func check(err error) {
    if err != nil {
        log.Fatalf("error: %v\n", err)
    }
}




func main() {
    mux := http.NewServeMux()
    
    apiCfg := &apiConfig{
        fileserverHits: atomic.Int32{},
    }
    
    handlerApp := http.StripPrefix("/app/", http.FileServer(http.Dir("./html/app")))

    mux.Handle("/app/", apiCfg.middlewareMetricsInc(handlerApp))
    mux.HandleFunc("/metrics/", apiCfg.handlerHits)
    mux.HandleFunc("/reset/", apiCfg.handlerReset)
    mux.HandleFunc("/healthz/", handlerHealthz)

    port := "8080"

    ex, err :=  os.Executable()
    check(err)
    filepathRoot := filepath.Dir(ex)

    srv := &http.Server{
        Addr: ":8080",
        Handler: mux,
    }

    log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)

    err = srv.ListenAndServe()
    check(err)

    fmt.Printf("works %v\n", srv)
}
