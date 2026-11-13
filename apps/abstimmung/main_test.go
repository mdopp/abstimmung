package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const wort = "bierdeckel"

// gast is one browser: it keeps the cookie the gate gave it.
type gast struct {
	t      *testing.T
	h      http.Handler
	cookie *http.Cookie
	ip     string
}

func neuerHandler(t *testing.T, w string) (http.Handler, *Speicher) {
	t.Helper()
	sp, err := LadeSpeicher(filepath.Join(t.TempDir(), "stimmen.json"))
	if err != nil {
		t.Fatal(err)
	}
	return handler(w, sp), sp
}

func (g *gast) tu(method, path, body string) *httptest.ResponseRecorder {
	g.t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if g.cookie != nil {
		req.AddCookie(g.cookie)
	}
	if g.ip != "" {
		req.Header.Set("X-Forwarded-For", g.ip)
	}
	rec := httptest.NewRecorder()
	g.h.ServeHTTP(rec, req)
	return rec
}

// eingeladen is a guest who opened the invite link.
func eingeladen(t *testing.T, h http.Handler) *gast {
	t.Helper()
	g := &gast{t: t, h: h}
	rec := g.tu("GET", "/?e="+wort, "")
	if len(rec.Result().Cookies()) != 1 {
		t.Fatalf("the invite link set no cookie: %d %s", rec.Code, rec.Body)
	}
	g.cookie = rec.Result().Cookies()[0]
	return g
}

// zettel is a full ballot: every chancellor in cell [m, k], then the changes.
func zettel(m, k int, name string, anders map[string][]int) string {
	z := map[string][]int{}
	for _, x := range kanzler {
		z[x.ID] = []int{m, k}
	}
	for id, c := range anders {
		if c == nil {
			delete(z, id)
		} else {
			z[id] = c
		}
	}
	raw, _ := json.Marshal(map[string]any{"zellen": z, "name": name})
	return string(raw)
}

func (g *gast) stimmt(body string) {
	g.t.Helper()
	if rec := g.tu("POST", "/api/stimme", body); rec.Code != 200 {
		g.t.Fatalf("vote refused: %d %s", rec.Code, rec.Body)
	}
}

func TestHealthNeedsNoInvite(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	if rec := (&gast{t: t, h: h}).tu("GET", "/healthz", ""); rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("healthz: %d %q", rec.Code, rec.Body)
	}
}

// Ablauf A-1
func TestWithoutTheInviteEveryPageIsTheOneLine(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	g := &gast{t: t, h: h}
	for _, c := range [][2]string{{"GET", "/"}, {"GET", "/ergebnis"}, {"POST", "/api/stimme"}, {"GET", "/nope"}, {"GET", "/?x=" + wort}} {
		body := ""
		if c[0] == "POST" {
			body = zettel(0, 0, "", nil)
		}
		rec := g.tu(c[0], c[1], body)
		if rec.Code != 403 || !strings.Contains(rec.Body.String(), zuText) {
			t.Errorf("%s %s: %d %q", c[0], c[1], rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "Kanzler") || len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s %s shows more than the line, or sets a cookie", c[0], c[1])
		}
	}
}

// Ablauf A-1: a box whose owner has entered no word lets nobody in, not
// even with an empty word in the link.
func TestWithoutAWordOnTheBoxNobodyGetsIn(t *testing.T) {
	h, _ := neuerHandler(t, "")
	g := &gast{t: t, h: h}
	for _, p := range []string{"/", "/?e=", "/?e=bierdeckel"} {
		if rec := g.tu("GET", p, ""); rec.Code != 403 || len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s: %d, cookies %v", p, rec.Code, rec.Result().Cookies())
		}
	}
}

// Ablauf A-2
func TestTheInviteLinkSetsTheCookieAndLeavesTheAddressBar(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	g := &gast{t: t, h: h}
	rec := g.tu("GET", "/?e="+wort, "")
	if rec.Code != 303 || rec.Header().Get("Location") != "/" {
		t.Fatalf("right word: %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	cs := rec.Result().Cookies()
	if len(cs) != 1 {
		t.Fatalf("cookies: %v", cs)
	}
	c := cs[0]
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge <= 0 {
		t.Errorf("cookie flags: %+v", c)
	}
	if strings.Contains(c.Value, wort) {
		t.Error("the cookie carries the word")
	}
	g.cookie = c
	if rec := g.tu("GET", "/", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Stimme abgeben") || !strings.Contains(rec.Body.String(), "Bierdeckel") {
		t.Fatalf("ballot with the cookie: %d", rec.Code)
	}
	// opening the link again keeps the voter: no second id
	if rec := g.tu("GET", "/?e="+wort, ""); rec.Code != 303 || len(rec.Result().Cookies()) != 0 {
		t.Errorf("second visit by the link: %d, cookies %v", rec.Code, rec.Result().Cookies())
	}
	// two guests are two voters
	if other := eingeladen(t, h); other.cookie.Value == c.Value {
		t.Error("two guests got the same id")
	}
}

// Ablauf A-1
func TestAWrongWordStaysOutside(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	g := &gast{t: t, h: h}
	for _, e := range []string{"", "falsch", wort + "x", wort[:len(wort)-1], strings.ToUpper(wort)} {
		rec := g.tu("GET", "/?e="+e, "")
		if rec.Code != 403 || len(rec.Result().Cookies()) != 0 || !strings.Contains(rec.Body.String(), zuText) {
			t.Errorf("word %q: %d, cookies %v", e, rec.Code, rec.Result().Cookies())
		}
	}
}

func TestGuessingTheWordIsSlowedDown(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	g := &gast{t: t, h: h, ip: "203.0.113.7"}
	for range fehlversucheProMinute {
		g.tu("GET", "/?e=falsch", "")
	}
	if rec := g.tu("GET", "/?e="+wort, ""); rec.Code != 429 || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("after %d wrong words the right one still gets in: %d", fehlversucheProMinute, rec.Code)
	}
	// another address is not held up
	if rec := (&gast{t: t, h: h, ip: "203.0.113.8"}).tu("GET", "/?e="+wort, ""); rec.Code != 303 {
		t.Fatalf("another address: %d", rec.Code)
	}
}

// Ablauf A-3
func TestANewWordEndsOldCookies(t *testing.T) {
	dir := t.TempDir()
	sp, _ := LadeSpeicher(filepath.Join(dir, "stimmen.json"))
	g := eingeladen(t, handler(wort, sp))
	g.h = handler("neuesjahr", sp)
	if rec := g.tu("GET", "/", ""); rec.Code != 403 {
		t.Fatalf("an old cookie after a new word: %d", rec.Code)
	}
	// and a cookie nobody signed is no cookie
	g.h = handler(wort, sp)
	id, _, _ := strings.Cut(g.cookie.Value, ".")
	for _, v := range []string{id, id + ".", id + "." + strings.Repeat("0", 64), "x." + (&Gate{Wort: wort}).mac("x")} {
		g.cookie = &http.Cookie{Name: cookieName, Value: v}
		if rec := g.tu("GET", "/", ""); rec.Code != 403 {
			t.Errorf("forged cookie %q: %d", v, rec.Code)
		}
	}
}

// Ablauf A-4
func TestOnlyAFullAndKnownBallotIsTaken(t *testing.T) {
	h, sp := neuerHandler(t, wort)
	g := eingeladen(t, h)
	bad := map[string]string{
		"nine":           zettel(0, 0, "", map[string][]int{"merz": nil}),
		"unknown id":     zettel(0, 0, "", map[string][]int{"merz": nil, "strauss": {0, 0}}),
		"eleven":         zettel(0, 0, "", map[string][]int{"strauss": {0, 0}}),
		"row 3":          zettel(0, 0, "", map[string][]int{"kohl": {3, 0}}),
		"column -1":      zettel(0, 0, "", map[string][]int{"kohl": {0, -1}}),
		"one number":     zettel(0, 0, "", map[string][]int{"kohl": {0}}),
		"three numbers":  zettel(0, 0, "", map[string][]int{"kohl": {0, 0, 0}}),
		"free text":      `{"zellen":{"adenauer":"der Beste"}}`,
		"extra field":    strings.Replace(zettel(0, 0, "", nil), `{"name"`, `{"kommentar":"frei","name"`, 1),
		"two objects":    zettel(0, 0, "", nil) + zettel(1, 1, "", nil),
		"not json":       "adenauer=0",
		"empty object":   "{}",
		"too large body": `{"name":"` + strings.Repeat("a", maxBody) + `"}`,
	}
	i := 0
	for name, body := range bad {
		i++
		g.ip = fmt.Sprintf("198.51.100.%d", i) // the limit is another test
		rec := g.tu("POST", "/api/stimme", body)
		var antwort struct{ Fehler string }
		json.Unmarshal(rec.Body.Bytes(), &antwort)
		if rec.Code != 400 || antwort.Fehler == "" {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}
	// a form post is not a ballot
	req := httptest.NewRequest("POST", "/api/stimme", strings.NewReader(zettel(0, 0, "", nil)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", "198.51.100.200")
	req.AddCookie(g.cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 415 {
		t.Errorf("form content type: %d", rec.Code)
	}
	if n := len(sp.Alle()); n != 0 {
		t.Fatalf("%d refused ballots were stored", n)
	}
	g.ip = "198.51.100.201"
	g.stimmt(zettel(2, 1, "", nil))
	if n := len(sp.Alle()); n != 1 {
		t.Fatalf("a good ballot: %d stored", n)
	}
}

func TestPostsAreLimitedPerAddress(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	g := eingeladen(t, h)
	g.ip = "203.0.113.9"
	for range postsProMinute {
		g.stimmt(zettel(0, 0, "", nil))
	}
	if rec := g.tu("POST", "/api/stimme", zettel(0, 0, "", nil)); rec.Code != 429 {
		t.Fatalf("post %d: %d", postsProMinute+1, rec.Code)
	}
	g.ip = "203.0.113.10"
	g.stimmt(zettel(0, 0, "", nil))
}

func TestTheLimitForgetsAfterItsWindow(t *testing.T) {
	now := time.Unix(1000, 0)
	l := NeuesLimit(2, time.Minute)
	l.jetzt = func() time.Time { return now }
	first, second, third := l.Zaehle("a"), l.Zaehle("a"), l.Zaehle("a")
	if !first || !second || third || !l.Voll("a") || l.Voll("b") {
		t.Fatal("two are allowed, the third is not, and b is another address")
	}
	now = now.Add(time.Minute)
	if l.Voll("a") || !l.Zaehle("a") {
		t.Fatal("a new window starts empty")
	}
}

// Ablauf A-5
func TestTheResultIsOnlyForWhoVoted(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	erste, zweiter := eingeladen(t, h), eingeladen(t, h)
	erste.stimmt(zettel(0, 0, "", nil))
	rec := zweiter.tu("GET", "/ergebnis", "")
	if rec.Code != 403 || strings.Contains(rec.Body.String(), "Adenauer") || strings.Contains(rec.Body.String(), "Stimmen") {
		t.Fatalf("result before voting: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "sobald du selbst abgestimmt hast") {
		t.Error("the refusal does not say what to do")
	}
	zweiter.stimmt(zettel(0, 0, "", nil))
	rec = zweiter.tu("GET", "/ergebnis?neu", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Adenauer") || !strings.Contains(rec.Body.String(), "Deine Stimme ist gespeichert.") ||
		!strings.Contains(rec.Body.String(), "Bisher haben 2 Personen abgestimmt.") {
		t.Fatalf("result after voting: %d", rec.Code)
	}
}

// Ablauf A-6
func TestAVoteCanBeChangedAndCountsOnce(t *testing.T) {
	h, sp := neuerHandler(t, wort)
	g := eingeladen(t, h)
	if rec := g.tu("GET", "/", ""); !strings.Contains(rec.Body.String(), `"zellen":{}`) || strings.Contains(rec.Body.String(), "Du hast schon abgestimmt") {
		t.Fatal("a first visit is not an empty ballot")
	}
	g.stimmt(zettel(0, 0, "Anna", nil))
	g.stimmt(zettel(1, 1, "", map[string][]int{"brandt": {2, 0}}))
	alle := sp.Alle()
	if len(alle) != 1 || alle[0].Name != "" || fmt.Sprint(alle[0].Zellen["brandt"], alle[0].Zellen["kohl"]) != "[2 0] [1 1]" {
		t.Fatalf("after a change: %+v", alle)
	}
	body := g.tu("GET", "/", "").Body.String()
	for _, want := range []string{`"brandt":[2,0]`, `"kohl":[1,1]`, `"abgestimmt":true`, "Du hast schon abgestimmt", ">Stimme ändern</button>"} {
		if !strings.Contains(body, want) {
			t.Errorf("the ballot of a voter who comes back lacks %s", want)
		}
	}
}

// Ablauf A-6: the votes outlive a restart of the program.
func TestVotesSurviveARestart(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "stimmen.json")
	sp, _ := LadeSpeicher(pfad)
	g := eingeladen(t, handler(wort, sp))
	g.stimmt(zettel(2, 2, "Bernd", nil))
	wieder, err := LadeSpeicher(pfad)
	if err != nil {
		t.Fatal(err)
	}
	g.h = handler(wort, wieder)
	if rec := g.tu("GET", "/ergebnis", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Bernd") {
		t.Fatalf("after a restart: %d", rec.Code)
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(pfad), ".*"))
	if len(left) != 0 {
		t.Errorf("temp files left beside the votes: %v", left)
	}
}

func TestABrokenFileIsNotOverwritten(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "stimmen.json")
	os.WriteFile(pfad, []byte(`{"stimmen": {`), 0o600)
	if _, err := LadeSpeicher(pfad); err == nil {
		t.Fatal("a file that is not JSON was read as an empty poll")
	}
	if _, err := LadeSpeicher(filepath.Join(t.TempDir(), "gibt", "es", "nicht.json")); err == nil {
		t.Fatal("a folder that cannot be written passed")
	}
}

func TestThePollHasARoof(t *testing.T) {
	sp, _ := LadeSpeicher(filepath.Join(t.TempDir(), "stimmen.json"))
	for i := range maxStimmen {
		sp.stimmen[fmt.Sprint(i)] = Stimme{}
	}
	if err := sp.Setze("neu", Stimme{}); err != errVoll {
		t.Fatalf("vote %d: %v", maxStimmen+1, err)
	}
	if err := sp.Setze("0", Stimme{Name: "x"}); err != nil {
		t.Fatalf("changing a vote in a full poll: %v", err)
	}
}

// Ablauf A-7
func TestEveryChancellorStandsWhereMostPutThem(t *testing.T) {
	var alle []Stimme
	for _, raw := range []string{
		zettel(0, 0, "", map[string][]int{"kohl": {1, 2}, "merz": {2, 2}}),
		zettel(0, 0, "", map[string][]int{"kohl": {1, 2}, "merz": {1, 1}}),
		zettel(2, 1, "", map[string][]int{"kohl": {1, 2}, "merz": {0, 1}}),
	} {
		var st Stimme
		json.Unmarshal([]byte(raw), &st)
		alle = append(alle, st)
	}
	e := werte(alle)
	find := func(id string) (cells []string, p Platz) {
		for m := range 3 {
			for k := range 3 {
				for _, x := range e.Felder[m][k] {
					if x.ID == id {
						cells, p = append(cells, fmt.Sprint(m, k)), x
					}
				}
			}
		}
		return
	}
	if cells, p := find("kohl"); fmt.Sprint(cells) != "[1 2]" || p.Anteil != 100 || p.Stimmen != 3 || p.Gleichstand {
		t.Errorf("kohl, three of three: %v %+v", cells, p)
	}
	if cells, p := find("adenauer"); fmt.Sprint(cells) != "[0 0]" || p.Anteil != 67 || p.Stimmen != 2 || p.Gleichstand {
		t.Errorf("adenauer, two of three: %v %+v", cells, p)
	}
	// three cells with one vote each: shown in each, and marked
	if cells, p := find("merz"); fmt.Sprint(cells) != "[0 1 1 1 2 2]" || p.Anteil != 33 || p.Stimmen != 1 || !p.Gleichstand {
		t.Errorf("merz, a tie: %v %+v", cells, p)
	}
	if e.Gesamt != 3 {
		t.Errorf("total %d", e.Gesamt)
	}
	if leer := werte(nil); leer.Gesamt != 0 || len(leer.Felder[0][0]) != 0 {
		t.Errorf("no votes: %+v", leer)
	}
}

// Ablauf A-7: the page says it in the family's words.
func TestTheResultPageShowsShareAndVotes(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	a, b, c := eingeladen(t, h), eingeladen(t, h), eingeladen(t, h)
	a.stimmt(zettel(0, 0, "", nil))
	b.stimmt(zettel(0, 0, "", nil))
	c.stimmt(zettel(2, 2, "", nil))
	body := c.tu("GET", "/ergebnis", "").Body.String()
	if !strings.Contains(body, "<b>Merkel</b><small>67 % · 2 Stimmen</small>") {
		t.Fatal("the result page lacks Merkel with 67 % and 2 Stimmen")
	}
	if strings.Contains(body, "33 %") {
		t.Error("the minority's cell is shown too")
	}
}

// Ablauf A-8
func TestTheParticipantLine(t *testing.T) {
	st := func(names ...string) (out []Stimme) {
		for _, n := range names {
			out = append(out, Stimme{Name: n})
		}
		return
	}
	for want, in := range map[string][]Stimme{
		"":                             nil,
		"Anna":                         st("Anna"),
		"1 ohne Namen":                 st(""),
		"Anna und Bernd":               st("Bernd", "Anna"),
		"anna, Bernd und Clara":        st("Clara", "Bernd", "anna"),
		"Anna und 1 ohne Namen":        st("", "Anna"),
		"Anna, Bernd und 3 ohne Namen": st("", "Bernd", "", "Anna", ""),
		"Anna, Anna und 2 ohne Namen":  st("Anna", "", "Anna", ""),
		"4 ohne Namen":                 st("", "", "", ""),
	} {
		if got := teilnehmer(in); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

// Ablauf A-8: on the page, names only, and never beside a choice.
func TestTheResultPageNamesWhoTookPart(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	for _, n := range []string{"", "Bernd", "", "Anna", ""} {
		eingeladen(t, h).stimmt(zettel(1, 1, n, nil))
	}
	g := eingeladen(t, h)
	g.stimmt(zettel(1, 1, "Zoe", nil))
	g.stimmt(zettel(1, 1, "", nil)) // a voter may take their name back
	body := g.tu("GET", "/ergebnis", "").Body.String()
	if !strings.Contains(body, `<p id="teilnehmer">Abgestimmt haben: Anna, Bernd und 4 ohne Namen</p>`) {
		t.Fatal("the participants line is missing or wrong")
	}
	if strings.Count(body, "Anna") != 1 || strings.Contains(body, "Zoe") {
		t.Error("a name shows up outside the participants line, or a cleared one stayed")
	}
}

// Ablauf A-8
func TestTheNameIsShortPlainText(t *testing.T) {
	h, sp := neuerHandler(t, wort)
	g := eingeladen(t, h)
	g.ip = "198.51.100.1"
	if rec := g.tu("POST", "/api/stimme", zettel(0, 0, strings.Repeat("ä", maxName+1), nil)); rec.Code != 400 || !strings.Contains(rec.Body.String(), "höchstens 40 Zeichen") {
		t.Fatalf("41 characters: %d %s", rec.Code, rec.Body)
	}
	g.stimmt(zettel(0, 0, strings.Repeat("ä", maxName), nil)) // 40 characters, 80 bytes
	for in, want := range map[string]string{
		"  Anna  ":               "Anna",
		"An\x00na\x07\x1b[31m":   "Anna[31m",
		"Anna\n\tMaria":          "Anna Maria",
		"Anna\u202egro\u2066":    "Annagro",
		" \t\n ":                 "",
		"Jürgen 🍺 O’Neil-Müller": "Jürgen 🍺 O’Neil-Müller",
	} {
		g.ip = "198.51.100." + fmt.Sprint(len(in)+10)
		g.stimmt(zettel(0, 0, in, nil))
		if got := sp.Alle()[0].Name; got != want {
			t.Errorf("name %q stored as %q, want %q", in, got, want)
		}
	}
	// blanks and control characters do not count towards the 40
	g.ip = "198.51.100.250"
	g.stimmt(zettel(0, 0, "  "+strings.Repeat("a", maxName)+"\x00  ", nil))
}

// Ablauf A-8
func TestANameIsEscapedWhereverItIsShown(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	g := eingeladen(t, h)
	boese := `<script>alert(1)</script>"'&`
	g.stimmt(zettel(0, 0, boese, nil))
	for _, p := range []string{"/ergebnis", "/"} {
		body := g.tu("GET", p, "").Body.String()
		if strings.Contains(body, "<script>alert") || strings.Contains(body, `</script>"`) {
			t.Errorf("%s shows the name as markup", p)
		}
		if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;&#34;&#39;&amp;") {
			t.Errorf("%s does not show the escaped name", p)
		}
	}
}

func TestNothingIsLoadedFromElsewhere(t *testing.T) {
	h, _ := neuerHandler(t, wort)
	g := eingeladen(t, h)
	g.stimmt(zettel(0, 0, "", nil))
	for _, p := range []string{"/", "/ergebnis"} {
		rec := g.tu("GET", p, "")
		body := rec.Body.String()
		for _, bad := range []string{"http://", "https://", "//fonts", "@import", "url("} {
			if strings.Contains(body, bad) {
				t.Errorf("%s contains %q", p, bad)
			}
		}
		if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'none'") || rec.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s: headers %v", p, rec.Header())
		}
	}
	if rec := g.tu("GET", "/nope", ""); rec.Code != 404 {
		t.Errorf("an unknown path for a guest: %d", rec.Code)
	}
}
