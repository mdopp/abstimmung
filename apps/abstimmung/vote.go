package main

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Kanzler is one of the ten the poll is about; the ids are the only words
// a vote may contain.
type Kanzler struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Kurz  string `json:"kurz"`
	Jahre string `json:"jahre"`
}

var kanzler = []Kanzler{
	{"adenauer", "Konrad Adenauer", "Adenauer", "1949–1963"},
	{"erhard", "Ludwig Erhard", "Erhard", "1963–1966"},
	{"kiesinger", "Kurt Georg Kiesinger", "Kiesinger", "1966–1969"},
	{"brandt", "Willy Brandt", "Brandt", "1969–1974"},
	{"schmidt", "Helmut Schmidt", "Schmidt", "1974–1982"},
	{"kohl", "Helmut Kohl", "Kohl", "1982–1998"},
	{"schroeder", "Gerhard Schröder", "Schröder", "1998–2005"},
	{"merkel", "Angela Merkel", "Merkel", "2005–2021"},
	{"scholz", "Olaf Scholz", "Scholz", "2021–2025"},
	{"merz", "Friedrich Merz", "Merz", "seit 2025"},
}

// the matrix: rows are the person, columns the chancellor, 0 = good … 2 = bad
var (
	stufen  = [3]string{"Guter", "Mittlerer", "Schlechter"}
	maxName = 40
)

// Stimme is one voter's ballot: for every chancellor the cell [row, column]
// and, if the voter wants, a name the others see among who took part.
type Stimme struct {
	Zellen map[string][]int `json:"zellen"`
	Name   string           `json:"name"`
}

// pruefe accepts exactly the ten known ids, each in a cell 0–2 × 0–2, and
// cleans the name. The messages are shown to the voter.
func pruefe(st *Stimme) error {
	if len(st.Zellen) != len(kanzler) {
		return errors.New("Bitte ordne alle zehn Kanzler ein.")
	}
	for _, k := range kanzler {
		z, ok := st.Zellen[k.ID]
		if !ok {
			return errors.New("Bitte ordne alle zehn Kanzler ein.")
		}
		if len(z) != 2 || z[0] < 0 || z[0] > 2 || z[1] < 0 || z[1] > 2 {
			return errors.New("Diese Stimme passt nicht zur Abstimmung. Bitte lade die Seite neu.")
		}
	}
	st.Name = saubererName(st.Name)
	if utf8.RuneCountInString(st.Name) > maxName {
		return fmt.Errorf("Der Name darf höchstens %d Zeichen lang sein.", maxName)
	}
	return nil
}

// saubererName: plain text on one line. Control characters and the marks
// that turn the writing direction go, white space becomes single blanks.
func saubererName(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' '
		case unicode.IsControl(r), r == utf8.RuneError, r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069:
			return -1
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

// Platz is a chancellor in the cell most voters chose for him or her.
type Platz struct {
	Kanzler
	Anteil      int // percent of all votes
	Stimmen     int
	Gleichstand bool // as many votes in another cell: shown in each of them
}

// Ergebnis is what a voter sees after voting: the matrix, how many voted,
// and who took part — never who voted for what.
type Ergebnis struct {
	Gesamt     int
	Felder     [3][3][]Platz
	Teilnehmer string
}

func werte(alle []Stimme) Ergebnis {
	e := Ergebnis{Gesamt: len(alle), Teilnehmer: teilnehmer(alle)}
	if e.Gesamt == 0 {
		return e
	}
	for _, k := range kanzler {
		var n [3][3]int
		for _, st := range alle {
			if z := st.Zellen[k.ID]; len(z) == 2 {
				n[z[0]][z[1]]++
			}
		}
		flach := slices.Concat(n[0][:], n[1][:], n[2][:])
		most, wieOft := slices.Max(flach), 0
		for _, c := range flach {
			if c == most {
				wieOft++
			}
		}
		for i, c := range flach {
			if c == most {
				e.Felder[i/3][i%3] = append(e.Felder[i/3][i%3], Platz{k, (most*100 + e.Gesamt/2) / e.Gesamt, most, wieOft > 1})
			}
		}
	}
	return e
}

// teilnehmer: „Anna, Bernd und 3 ohne Namen“, the names in the alphabet's
// order so the line says nothing about who came when.
func teilnehmer(alle []Stimme) string {
	var teile []string
	ohne := 0
	for _, st := range alle {
		if st.Name == "" {
			ohne++
		} else {
			teile = append(teile, st.Name)
		}
	}
	slices.SortFunc(teile, func(a, b string) int {
		return cmp.Or(cmp.Compare(strings.ToLower(a), strings.ToLower(b)), cmp.Compare(a, b))
	})
	if ohne > 0 {
		teile = append(teile, fmt.Sprintf("%d ohne Namen", ohne))
	}
	switch len(teile) {
	case 0:
		return ""
	case 1:
		return teile[0]
	}
	return strings.Join(teile[:len(teile)-1], ", ") + " und " + teile[len(teile)-1]
}
