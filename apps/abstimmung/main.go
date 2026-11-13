// abstimmung: a poll that came from a beer coaster. Invited guests place the
// ten German chancellors in a 3×3 matrix (how good as a person, how good as
// a chancellor) and see the result once they have voted.
//
// The route is public and has no Authelia in front (auth: none), so the app
// is its own gate: a visitor needs the invite link with the owner's word
// (gate.go). There are no accounts; one browser is one voter, by a cookie.
// Standard library only; the votes are one JSON file under /data (store.go).
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

// handler is the whole program: /healthz for the box, everything else
// behind the gate.
func handler(wort string, sp *Speicher) http.Handler {
	app := &App{Speicher: sp, Post: NeuesLimit(postsProMinute, time.Minute)}
	gate := &Gate{Wort: wort, Fehlversuche: NeuesLimit(fehlversucheProMinute, time.Minute)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok\n")) })
	mux.Handle("/", gate.Schuetze(app.Routen()))
	return mux
}

func main() {
	sp, err := LadeSpeicher(filepath.Join(env("DATA", "/data"), "stimmen.json"))
	if err != nil {
		log.Fatalf("abstimmung: %v", err)
	}
	wort := os.Getenv("EINLADUNG")
	if wort == "" {
		log.Print("abstimmung: no invite word (EINLADUNG) is set, every page answers 403 until the owner enters one")
	}
	srv := &http.Server{
		Addr:              env("LISTEN", ":8080"),
		Handler:           handler(wort, sp),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	log.Printf("abstimmung listens on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
