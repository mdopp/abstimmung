package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	cookieName            = "abstimmung"
	cookieTage            = 365
	fehlversucheProMinute = 10
	zuText                = "Diese Abstimmung ist nur mit Einladungslink offen."
	zuVieleText           = "Zu viele Versuche. Bitte warte eine Minute."
)

// Gate lets in only who came by the invite link https://<host>/?e=<word>.
// The right word sets a cookie "<voter id>.<MAC>", the MAC keyed by the
// word: the cookie says both who the voter is and that the invite was
// accepted, without a session store, and a new word makes every old cookie
// worthless. Without a word (the owner has not entered one) nobody gets in.
type Gate struct {
	Wort         string
	Fehlversuche *Limit // wrong words per address, so the word cannot be guessed by trying
}

type waehlerKey struct{}

// Waehler is the voter id the gate found in the request's cookie.
func Waehler(r *http.Request) string {
	id, _ := r.Context().Value(waehlerKey{}).(string)
	return id
}

// Schuetze wraps the pages: a request with ?e= is an invite attempt, any
// other needs a valid cookie; all else is the one German line with 403.
func (g *Gate) Schuetze(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kopf(w)
		if r.Method == http.MethodGet && r.URL.Query().Has("e") {
			g.einladung(w, r)
			return
		}
		id := g.waehler(r)
		if id == "" {
			zeile(w, http.StatusForbidden, zuText)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), waehlerKey{}, id)))
	})
}

func (g *Gate) einladung(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if g.Fehlversuche.Voll(ip) {
		zeile(w, http.StatusTooManyRequests, zuVieleText)
		return
	}
	if g.Wort == "" || !gleich(r.URL.Query().Get("e"), g.Wort) {
		g.Fehlversuche.Zaehle(ip)
		zeile(w, http.StatusForbidden, zuText)
		return
	}
	if g.waehler(r) == "" { // a guest who opens the link again keeps their id, and so their vote
		id := neueID()
		http.SetCookie(w, &http.Cookie{
			Name: cookieName, Value: id + "." + g.mac(id), Path: "/", MaxAge: cookieTage * 24 * 3600,
			HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		})
	}
	http.Redirect(w, r, "/", http.StatusSeeOther) // the word leaves the address bar
}

// waehler is the voter id of a cookie this word signed, or "".
func (g *Gate) waehler(r *http.Request) string {
	c, err := r.Cookie(cookieName)
	if err != nil || g.Wort == "" {
		return ""
	}
	id, mac, ok := strings.Cut(c.Value, ".")
	if !ok || len(id) != 32 || !istHex(id) || !hmac.Equal([]byte(mac), []byte(g.mac(id))) {
		return ""
	}
	return id
}

func (g *Gate) mac(id string) string {
	key := sha256.Sum256([]byte("abstimmung cookie\x00" + g.Wort))
	h := hmac.New(sha256.New, key[:])
	h.Write([]byte(id))
	return hex.EncodeToString(h.Sum(nil))
}

// gleich compares in constant time; hashing first hides the lengths too.
func gleich(a, b string) bool {
	x, y := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(x[:], y[:]) == 1
}

func neueID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // no randomness: no voter ids
	}
	return hex.EncodeToString(b)
}

func istHex(s string) bool {
	return strings.Trim(s, "0123456789abcdef") == ""
}

// kopf: nothing is loaded from elsewhere, nothing is cached, and no
// address (it may carry the word) is passed on as a referrer.
func kopf(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Cache-Control", "no-store")
}

// clientIP is the address Caddy saw: it appends the peer to
// X-Forwarded-For, so the last entry is the one a client cannot forge.
// Only Caddy reaches the port (published on loopback).
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(xff[strings.LastIndexByte(xff, ',')+1:])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Limit counts per address in a fixed window; the window's end forgets
// everything, so the map cannot grow without bound.
type Limit struct {
	max     int
	fenster time.Duration
	jetzt   func() time.Time
	mu      sync.Mutex
	start   time.Time
	n       map[string]int
}

func NeuesLimit(max int, fenster time.Duration) *Limit {
	return &Limit{max: max, fenster: fenster, jetzt: time.Now, n: map[string]int{}}
}

func (l *Limit) lauf() {
	if now := l.jetzt(); now.Sub(l.start) >= l.fenster {
		l.start, l.n = now, map[string]int{}
	}
}

// Voll: this address has used up its window.
func (l *Limit) Voll(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lauf()
	return l.n[ip] >= l.max
}

// Zaehle counts one more and says whether it was still within the limit.
func (l *Limit) Zaehle(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lauf()
	l.n[ip]++
	return l.n[ip] <= l.max
}
