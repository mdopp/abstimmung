package main

import "html/template"

// The pages. Everything is inline (no font, script or image from anywhere
// else: nobody learns that a guest opened the poll), light and dark follow
// the device, and whatever can be touched is at least 44px high.

const kopfHTML = `<!doctype html>
<html lang="de">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="light dark">
<meta name="robots" content="noindex">
<title>Mensch und Kanzler</title>
<style>
:root{--bg:#f6f4ef;--fl:#fffdf8;--tx:#23201b;--lei:#6b655b;--li:#d9d3c6;--ak:#7a4a12;--akt:#fff;--chip:#efe7d6;--err:#a12a1c;--ok:#2d6a3e}
@media (prefers-color-scheme:dark){:root{--bg:#181715;--fl:#22201d;--tx:#ece7dd;--lei:#a59e91;--li:#3a3732;--ak:#e0a458;--akt:#1b1711;--chip:#33302a;--err:#f08b7c;--ok:#8fcf9f}}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--tx);font:16px/1.5 system-ui,-apple-system,"Segoe UI",sans-serif}
main{max-width:46rem;margin:0 auto;padding:1.25rem .75rem 3rem}
h1{font-size:1.6rem;line-height:1.2;margin:.5rem 0 .5rem}
h2{font-size:1rem;margin:1.25rem 0 .4rem}
p{margin:.5rem 0}
.leise{color:var(--lei);font-size:.95rem}
.reihe{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.4rem}
.spalten{position:sticky;top:0;background:var(--bg);padding:.4rem 0;z-index:1}
.spalten div{font-size:.8rem;font-weight:600;text-align:center;color:var(--lei);hyphens:auto}
.feld{background:var(--fl);border:1px solid var(--li);border-radius:.6rem;padding:.35rem;min-height:5.5rem;display:flex;flex-direction:column;gap:.35rem}
.chips{display:flex;flex-direction:column;gap:.35rem}
button,select,input,.knopf{font:inherit;min-height:44px;border-radius:.5rem}
.chip{border:1px solid var(--li);background:var(--chip);color:var(--tx);padding:.3rem .45rem;text-align:left;display:flex;justify-content:space-between;align-items:center;gap:.25rem;width:100%;cursor:pointer;overflow-wrap:anywhere}
.chip span{color:var(--lei)}
select{width:100%;border:1px dashed var(--li);background:transparent;color:var(--lei);padding:0 .3rem;margin-top:auto}
select:disabled{opacity:.45}
label{display:block;font-weight:600;margin:1.25rem 0 .3rem}
input[type=text]{width:100%;border:1px solid var(--li);background:var(--fl);color:var(--tx);padding:0 .7rem}
.haupt,.knopf{display:inline-flex;align-items:center;justify-content:center;border:0;background:var(--ak);color:var(--akt);font-weight:600;padding:0 1.25rem;text-decoration:none;cursor:pointer}
.haupt{width:100%;margin-top:1rem}
.haupt:disabled{opacity:.45;cursor:not-allowed}
.zweit{background:transparent;color:var(--ak);border:1px solid var(--ak)}
#status{min-height:1.5rem;margin:.6rem 0 0}
.fehler{color:var(--err)}.gut{color:var(--ok)}
.platz{background:var(--chip);border-radius:.5rem;padding:.3rem .45rem;overflow-wrap:anywhere}
.platz b{display:block}
.platz small{color:var(--lei)}
:focus-visible{outline:3px solid var(--ak);outline-offset:2px}
@media (min-width:36rem){.spalten div{font-size:.95rem}.feld{padding:.5rem}}
</style>
</head>
<body>
<main>
`

const spaltenHTML = `<div class="reihe spalten" aria-hidden="true">{{range .Stufen}}<div>{{.}} Kanzler</div>{{end}}</div>`

var wahlzettelTmpl = template.Must(template.New("wahlzettel").Parse(kopfHTML + `<h1>Mensch und Kanzler</h1>
<p>Entstanden auf einem Bierdeckel: Wie gut war jemand als Mensch, und wie gut als Kanzler? Ordne jeden der zehn Bundeskanzler in ein Feld ein. Das Ergebnis siehst du, sobald du abgestimmt hast.</p>
{{if .Abgestimmt}}<p class="leise">Du hast schon abgestimmt. Hier kannst du deine Stimme ändern. <a href="/ergebnis">Zum Ergebnis</a></p>{{end}}
<noscript><p class="fehler">Zum Abstimmen braucht diese Seite JavaScript.</p></noscript>
` + spaltenHTML + `
{{range $m, $mensch := .Stufen}}<h2>{{$mensch}} Mensch</h2>
<div class="reihe">{{range $k, $kanzler := $.Stufen}}<div class="feld" data-m="{{$m}}" data-k="{{$k}}"><div class="chips"></div><select aria-label="Kanzler hinzufügen: {{$mensch}} Mensch, {{$kanzler}} Kanzler"><option value="">+ Kanzler</option></select></div>{{end}}</div>
{{end}}
<p id="offen" class="leise" aria-live="polite"></p>
<label for="name">Dein Name (freiwillig)</label>
<input type="text" id="name" maxlength="{{.MaxName}}" autocomplete="given-name" value="{{.Name}}">
<p class="leise">Mit Namen sehen die anderen, dass du mitgemacht hast, aber nie, wie du abgestimmt hast.</p>
<button class="haupt" id="abgeben" disabled>{{if .Abgestimmt}}Stimme ändern{{else}}Stimme abgeben{{end}}</button>
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
<p><a class="knopf zweit" href="/">Meine Stimme ändern</a></p>
</main>
</body>
</html>
`))

var erstAbstimmenTmpl = template.Must(template.New("erst").Parse(kopfHTML + `<h1>Erst abstimmen</h1>
<p>Das Ergebnis siehst du, sobald du selbst abgestimmt hast.</p>
<p><a class="knopf" href="/">Zur Abstimmung</a></p>
</main>
</body>
</html>
`))
