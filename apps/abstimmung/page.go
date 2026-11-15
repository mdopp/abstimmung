package main

import (
	"html/template"

	"github.com/mdopp/gestaltung"
)

// The pages. Everything is inline (no font, script or image from anywhere
// else: nobody learns that a guest opened the poll), light and dark follow
// the device, and whatever can be touched is at least 44px high.

// kopfHTML is the head of every page: the household's shared look
// (mdopp/gestaltung, embedded at build time) and the few rules only the
// poll needs, written in its tokens. No colour literal here: the check in
// scripts/farben.py fails CI on one.
var kopfHTML = `<!doctype html>
<html lang="de">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light dark">
<meta name="robots" content="noindex">
<title>Mensch und Kanzler</title>
<style>
` + gestaltung.CSS + `
body{margin:0;background:var(--bg)}
.page{padding-top:var(--space-2)}
.page h2{font-size:var(--text-m);font-weight:700;letter-spacing:0;text-transform:none;color:var(--ink);margin:var(--space-5) 0 var(--space-2)}
.page .muted{font-size:var(--text-s)}
.reihe{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:var(--space-2)}
.spalten{position:sticky;top:0;background:var(--bg);padding:var(--space-2) 0;z-index:1}
.spalten div{font-size:var(--text-s);font-weight:700;text-align:center;hyphens:auto}
.feld{background:var(--card);border:1px solid var(--edge);border-radius:var(--radius-s);padding:var(--space-1);min-height:5.5rem;display:flex;flex-direction:column;gap:var(--space-1)}
.chips{display:flex;flex-direction:column;gap:var(--space-1)}
select,input{font:inherit;min-height:var(--touch);border-radius:var(--radius-s)}
.chip{font:inherit;min-height:var(--touch);border:1px solid var(--edge);border-radius:var(--radius-s);background:var(--accent-soft);color:var(--ink);padding:var(--space-1) var(--space-2);text-align:left;display:flex;justify-content:space-between;align-items:center;gap:var(--space-1);width:100%;cursor:pointer;overflow-wrap:anywhere}
.chip span{color:var(--muted)}
select{width:100%;border:1px dashed var(--edge);background:transparent;color:var(--muted);padding:0 var(--space-1);margin-top:auto}
select:disabled{opacity:.45}
label{display:block;font-weight:600;margin:var(--space-5) 0 var(--space-1)}
input[type=text]{width:100%;border:1px solid var(--edge);background:var(--card);color:var(--ink);padding:0 var(--space-3)}
.haupt{width:100%;margin-top:var(--space-4)}
#status{min-height:1.5rem;margin:var(--space-2) 0 0}
.fehler{color:var(--bad)}.gut{color:var(--ok)}
.platz{background:var(--accent-soft);border-radius:var(--radius-s);padding:var(--space-1) var(--space-2);overflow-wrap:anywhere}
.platz b{display:block}
.platz small{color:var(--muted)}
.chip:focus-visible,select:focus-visible,input:focus-visible{outline:2px solid var(--accent);outline-offset:2px}
@media (min-width:36rem){.spalten div{font-size:var(--text-m)}.feld{padding:var(--space-2)}}
</style>
</head>
<body>
<main class="page">
`

const spaltenHTML = `<div class="reihe spalten" aria-hidden="true">{{range .Stufen}}<div>{{.}} Kanzler</div>{{end}}</div>`

var wahlzettelTmpl = template.Must(template.New("wahlzettel").Parse(kopfHTML + `<h1>Mensch und Kanzler</h1>
<p>Entstanden auf einem Bierdeckel: Wie gut war jemand als Mensch, und wie gut als Kanzler? Ordne jeden der zehn Bundeskanzler in ein Feld ein. Das Ergebnis siehst du, sobald du abgestimmt hast.</p>
{{if .Abgestimmt}}<p class="muted">Du hast schon abgestimmt. Hier kannst du deine Stimme ändern. <a href="/ergebnis">Zum Ergebnis</a></p>{{end}}
<noscript><p class="fehler">Zum Abstimmen braucht diese Seite JavaScript.</p></noscript>
` + spaltenHTML + `
{{range $m, $mensch := .Stufen}}<h2>{{$mensch}} Mensch</h2>
<div class="reihe">{{range $k, $kanzler := $.Stufen}}<div class="feld" data-m="{{$m}}" data-k="{{$k}}"><div class="chips"></div><select aria-label="Kanzler hinzufügen: {{$mensch}} Mensch, {{$kanzler}} Kanzler"><option value="">+ Kanzler</option></select></div>{{end}}</div>
{{end}}
<p id="offen" class="muted" aria-live="polite"></p>
<label for="name">Dein Name (freiwillig)</label>
<input type="text" id="name" maxlength="{{.MaxName}}" autocomplete="given-name" value="{{.Name}}">
<p class="muted">Mit Namen sehen die anderen, dass du mitgemacht hast, aber nie, wie du abgestimmt hast.</p>
<button class="button haupt" id="abgeben" disabled>{{if .Abgestimmt}}Stimme ändern{{else}}Stimme abgeben{{end}}</button>
<p id="status" role="status" aria-live="polite"></p>
</main>
<script>
"use strict";
{
const D = {{.Daten}};
const wahl = Object.assign({}, D.zellen);
const felder = [...document.querySelectorAll(".feld")];
const knopf = document.getElementById("abgeben"), status = document.getElementById("status");
const knopfText = knopf.textContent;
let sendet = false;

function melde(text, art) { status.textContent = text; status.className = art || ""; }

function zeichne() {
  const offen = D.kanzler.filter(k => !wahl[k.id]);
  for (const feld of felder) {
    const m = +feld.dataset.m, k = +feld.dataset.k;
    const chips = feld.querySelector(".chips"), sel = feld.querySelector("select");
    chips.replaceChildren(...D.kanzler.filter(x => wahl[x.id] && wahl[x.id][0] === m && wahl[x.id][1] === k).map(x => {
      const b = document.createElement("button");
      b.type = "button"; b.className = "chip"; b.dataset.id = x.id;
      b.setAttribute("aria-label", x.name + " entfernen");
      b.append(x.kurz);
      const kreuz = document.createElement("span"); kreuz.textContent = "✕"; kreuz.setAttribute("aria-hidden", "true");
      b.append(kreuz);
      return b;
    }));
    sel.replaceChildren(new Option("+ Kanzler", ""), ...offen.map(x => new Option(x.name + " (" + x.jahre + ")", x.id)));
    sel.disabled = offen.length === 0;
  }
  document.getElementById("offen").textContent = offen.length === 0 ? "Alle zehn sind eingeordnet." : "Noch " + offen.length + " von " + D.kanzler.length + " offen.";
  knopf.disabled = sendet || offen.length > 0;
}

for (const feld of felder) {
  feld.querySelector("select").addEventListener("change", e => {
    if (e.target.value) { wahl[e.target.value] = [+feld.dataset.m, +feld.dataset.k]; melde(""); zeichne(); }
  });
  feld.querySelector(".chips").addEventListener("click", e => {
    const chip = e.target.closest(".chip");
    if (chip) { delete wahl[chip.dataset.id]; melde(""); zeichne(); }
  });
}

knopf.addEventListener("click", async () => {
  sendet = true; knopf.textContent = "Wird gespeichert …"; zeichne(); melde("");
  const ab = new AbortController(), uhr = setTimeout(() => ab.abort(), 10000);
  let text = "Das hat nicht geklappt. Bitte prüfe deine Verbindung und versuche es noch einmal.";
  try {
    const r = await fetch("/api/stimme", {method: "POST", headers: {"Content-Type": "application/json"}, signal: ab.signal,
      body: JSON.stringify({zellen: wahl, name: document.getElementById("name").value})});
    if (r.ok) { melde("Gespeichert. Das Ergebnis kommt …", "gut"); location.assign("/ergebnis?neu"); return; }
    if (r.status === 403) text = "Dein Einladungslink gilt nicht mehr. Bitte öffne ihn noch einmal.";
    else { const j = await r.json().catch(() => null); if (j && j.fehler) text = j.fehler; }
  } catch (e) { /* the sentence above */ } finally { clearTimeout(uhr); }
  sendet = false; knopf.textContent = knopfText; zeichne(); melde(text, "fehler");
});

zeichne();
}
</script>
</body>
</html>
`))

var ergebnisTmpl = template.Must(template.New("ergebnis").Parse(kopfHTML + `<h1>Ergebnis</h1>
{{if .Neu}}<p class="gut" role="status">Deine Stimme ist gespeichert.</p>{{end}}
<p>{{if eq .E.Gesamt 1}}Bisher hat 1 Person abgestimmt.{{else}}Bisher haben {{.E.Gesamt}} Personen abgestimmt.{{end}} Jeder Kanzler steht in dem Feld, das die meisten gewählt haben.</p>
` + spaltenHTML + `
{{range $m, $mensch := .Stufen}}<h2>{{$mensch}} Mensch</h2>
<div class="reihe">{{range index $.E.Felder $m}}<div class="feld">{{range .}}<div class="platz"><b>{{.Kurz}}</b><small>{{.Anteil}} % · {{.Stimmen}} {{if eq .Stimmen 1}}Stimme{{else}}Stimmen{{end}}{{if .Gleichstand}} · Gleichstand{{end}}</small></div>{{end}}</div>{{end}}</div>
{{end}}
{{if .E.Teilnehmer}}<p id="teilnehmer">Abgestimmt haben: {{.E.Teilnehmer}}</p>{{end}}
<p><a class="button quiet" href="/">Meine Stimme ändern</a></p>
</main>
</body>
</html>
`))

var erstAbstimmenTmpl = template.Must(template.New("erst").Parse(kopfHTML + `<h1>Erst abstimmen</h1>
<p>Das Ergebnis siehst du, sobald du selbst abgestimmt hast.</p>
<p><a class="button" href="/">Zur Abstimmung</a></p>
</main>
</body>
</html>
`))
