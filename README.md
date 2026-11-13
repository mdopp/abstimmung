# abstimmung

A small poll for the household's Laib box, born on a beer coaster: how good
was each German chancellor as a person, and how good as a chancellor? Invited
guests place the ten chancellors in a 3×3 matrix and see the result once they
have voted. Served at `abstimmung.<domain>`; made in the Werkstatt
(`laib-app new abstimmung`).

- `apps/abstimmung/` — the program (Go, standard library only), its tests
  and its `Containerfile`; its README says what it does and its contract
  with the template.
- `templates/abstimmung/` — the Laib template: `laib.yml`, the Quadlet unit,
  `values.example.yml`, `news`.
- `.github/workflows/apps.yml` — tests the app, lints the template, runs
  the gates, and from main publishes `ghcr.io/mdopp/abstimmung`, signed
  with cosign.
- `scripts/` — the household's gates (line cap, ratchet, pins, Abläufe),
  copies of laib's scaffold; `docs/ABLAEUFE.md` holds the scenarios.

The box knows this repo as a template registry (box.yml `registries`);
`laib template sync` pulls it, `laib propose --add abstimmung` installs it
as a proposal the owner approves.
