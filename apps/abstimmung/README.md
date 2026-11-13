# abstimmung

A poll that came from a beer coaster: invited guests place the ten German
chancellors in a 3×3 matrix (rows: how good as a person, columns: how good
as a chancellor) and see the result once they have voted.

- Code: `main.go` (wiring), `gate.go` (the invite gate, the cookie, the
  limits), `vote.go` (the chancellors, validation, the result), `store.go`
  (the JSON file), `web.go` (the handlers), `page.go` (the pages). Standard
  library only. Tests: `go test ./...`; the scenarios they prove are in
  `../../docs/ABLAEUFE.md`.
- Image: `ghcr.io/mdopp/abstimmung`, built and signed by CI on every push to main.
- Laib template: `../../templates/abstimmung/` (route `abstimmung.<domain>`).

## The gate

The route is public and `auth: none`: the guests have no login on the box,
so the app refuses by itself.

- The invite link is `https://abstimmung.<domain>/?e=<word>`. The word is
  `EINLADUNG`, the owner's key `ABSTIMMUNG_EINLADUNG`. It is compared in
  constant time; ten wrong words from one address stop further tries for a
  minute.
- The right word sets the cookie `abstimmung` (HttpOnly, Secure,
  SameSite=Lax, a year): a random voter id and a MAC over it keyed by the
  word. Then the browser is sent to `/`, so the word leaves the address bar.
  The cookie is both the voter and the proof that the invite was accepted;
  there is no session store. A new word makes every old cookie invalid.
- Without a valid cookie every path except `/healthz` answers 403 with one
  line: „Diese Abstimmung ist nur mit Einladungslink offen.“ So does a box
  whose owner has not entered a word yet. `/ergebnis` is the template's
  `loginProbe`.

## The paths

| Path | What |
|---|---|
| `GET /healthz` | `ok`, for the box; no invite needed |
| `GET /?e=<word>` | the invite: cookie, then 303 to `/` |
| `GET /` | the ballot, prefilled for a voter who comes back |
| `POST /api/stimme` | the ballot as JSON `{"zellen": {"<id>": [row, column], …}, "name": "…"}`: exactly the ten known ids, both numbers 0–2, no other field, at most 4 KB, ten posts per address and minute. A second post replaces the first |
| `GET /ergebnis` | the result; 403 for a visitor who has not voted |

The result puts each chancellor in the cell most voters chose, with the
share of all votes and the number of votes; with a tie the chancellor is
shown in each of the tied cells, marked „Gleichstand“. Below it: who took
part, by the name a voter chose to give („Anna, Bernd und 3 ohne Namen“),
never who voted for what. The name is optional, at most 40 characters,
without control characters, and escaped by `html/template` wherever shown.

## Data

One file, `$DATA/stimmen.json` (`/data` in the container, the template's
`${DATA_DIR}/abstimmung`): the votes by voter id, each with its optional
name. Written to a temp file and renamed; a file that cannot be read stops
the program instead of being overwritten. At most 5,000 votes. Nothing
leaves the box, and the pages load nothing from elsewhere.

One vote per browser, not per person: a guest with two browsers, or one
who deletes the cookie and opens the link again, votes twice. That is the
price of having no accounts.

## Environment

`EINLADUNG` (the invite word), `DATA` (default `/data`), `LISTEN` (default `:8080`).
