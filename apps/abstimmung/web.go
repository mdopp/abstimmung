package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	postsProMinute = 10
	maxBody        = 4 << 10 // a full ballot is about 300 bytes
)

// App is the poll behind the gate: the ballot, the vote, the result.
type App struct {
	Speicher *Speicher
	Post     *Limit
}

func (a *App) Routen() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.wahlzettel)
	mux.HandleFunc("GET /ergebnis", a.ergebnis)
	mux.HandleFunc("POST /api/stimme", a.stimme)
	return mux
}

// wahlzettel is the ballot, prefilled for a voter who comes back.
func (a *App) wahlzettel(w http.ResponseWriter, r *http.Request) {
	st, da := a.Speicher.Hole(Waehler(r))
	zellen := st.Zellen
	if zellen == nil {
		zellen = map[string][]int{}
	}
	zeige(w, http.StatusOK, wahlzettelTmpl, map[string]any{
		"Stufen": stufen, "Abgestimmt": da, "Name": st.Name, "MaxName": maxName,
		"Daten": map[string]any{"kanzler": kanzler, "zellen": zellen, "abgestimmt": da},
	})
}

// ergebnis is only for who has voted: 403 otherwise, so nobody reads the
// others' choice before making their own.
func (a *App) ergebnis(w http.ResponseWriter, r *http.Request) {
	if _, da := a.Speicher.Hole(Waehler(r)); !da {
		zeige(w, http.StatusForbidden, erstAbstimmenTmpl, nil)
		return
	}
	zeige(w, http.StatusOK, ergebnisTmpl, map[string]any{
		"Stufen": stufen, "E": werte(a.Speicher.Alle()), "Neu": r.URL.Query().Has("neu"),
	})
}

// stimme takes a ballot as JSON, nothing else: the known fields, one
// object, a small body.
func (a *App) stimme(w http.ResponseWriter, r *http.Request) {
	if !a.Post.Zaehle(clientIP(r)) {
		fehler(w, http.StatusTooManyRequests, zuVieleText)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fehler(w, http.StatusUnsupportedMediaType, "Diese Stimme passt nicht zur Abstimmung. Bitte lade die Seite neu.")
		return
	}
	var st Stimme
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&st); err != nil || dec.More() {
		fehler(w, http.StatusBadRequest, "Diese Stimme passt nicht zur Abstimmung. Bitte lade die Seite neu.")
		return
	}
	if err := pruefe(&st); err != nil {
		fehler(w, http.StatusBadRequest, err.Error())
		return
	}
	switch err := a.Speicher.Setze(Waehler(r), st); {
	case errors.Is(err, errVoll):
		fehler(w, http.StatusServiceUnavailable, "Die Abstimmung nimmt keine neuen Stimmen mehr an.")
	case err != nil:
		log.Printf("abstimmung: the vote was not stored: %v", err)
		fehler(w, http.StatusInternalServerError, "Die Stimme konnte nicht gespeichert werden. Bitte versuche es gleich noch einmal.")
	default:
		antwort(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func antwort(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// fehler is a refusal in words the voter can act on.
func fehler(w http.ResponseWriter, status int, text string) {
	antwort(w, status, map[string]string{"fehler": text})
}

func zeige(w http.ResponseWriter, status int, t *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.Execute(w, data); err != nil {
		log.Printf("abstimmung: %s: %v", t.Name(), err)
	}
}

// zeile is a page of one sentence: all a visitor without an invite gets.
func zeile(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, `<!doctype html><html lang="de"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light dark"><title>Abstimmung</title><body style="font:1.05rem/1.5 system-ui,sans-serif;margin:3rem auto;max-width:28rem;padding:0 1rem"><p>`+template.HTMLEscapeString(text)+`</p>`)
}
