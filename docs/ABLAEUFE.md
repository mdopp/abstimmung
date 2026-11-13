# Abläufe — what the family sees these apps do

Every program of the household writes down its behaviour as numbered
scenarios („Abläufe“, owner 2026-11-12), in plain German, as the family or
the owner would tell it: what they see and do, not how the code does it.
Each scenario is proven by at least one test that names its id;
`scripts/ablaeufe.py` fails CI otherwise.

The format: a heading that starts with the id and one sentence, then two to
five short lines Wenn/Dann (also „Und“). The id is `<PREFIX>-<n>`, one
prefix per repo, numbers never reused. A test names it in a comment or its
name: `// Ablauf A-1` (`# Ablauf A-1, A-2` for several, `ablauf:A-1` in a
test title).

```
### A-1 Ein Satz, was passiert
Wenn jemand …
Dann sieht er …
```

A feature PR adds or changes its scenario and the test naming it. A
scenario no test proves yet is in `scripts/ablaeufe-baseline.txt`, which
only shrinks (laib's `docs/ABLAEUFE.md` is the example).

## Die Apps

### A-1 Ohne Einladungslink bleibt die Abstimmung zu
Wenn jemand die Adresse der Abstimmung ohne Einladungslink öffnet, oder mit einem falschen Wort,
Dann sieht er nur den Satz „Diese Abstimmung ist nur mit Einladungslink offen.“
Und das gilt für jede Seite, auch für das Ergebnis.

### A-2 Der Einladungslink öffnet die Abstimmung
Wenn ein Gast den Einladungslink öffnet,
Dann sieht er die Abstimmung, und das Einladungswort steht nicht mehr in der Adresszeile.
Und sein Browser bleibt eingeladen, auch wenn er die Seite später ohne den Link öffnet.

### A-3 Ein neues Einladungswort schließt alte Gäste aus
Wenn der Besitzer ein neues Einladungswort einträgt,
Dann gelten die alten Einladungslinks nicht mehr.
Und wer mit dem alten Wort eingeladen war, braucht den neuen Link.

### A-4 Abstimmen geht nur mit allen zehn Kanzlern
Wenn ein Gast noch nicht alle zehn Kanzler in ein Feld gelegt hat,
Dann kann er seine Stimme nicht abgeben.
Und etwas anderes als die zehn Kanzler in den neun Feldern nimmt die Abstimmung nicht an.

### A-5 Das Ergebnis sieht nur, wer abgestimmt hat
Wenn ein Gast das Ergebnis öffnet, bevor er abgestimmt hat,
Dann sieht er nur, dass er erst abstimmen muss.
Und sobald er abgestimmt hat, sieht er das Ergebnis sofort.

### A-6 Wer abgestimmt hat, kann seine Stimme ändern
Wenn ein Gast die Abstimmung noch einmal öffnet,
Dann liegen die Kanzler dort, wo er sie hingelegt hat.
Und gibt er sie geändert ab, zählt nur die neue Stimme.

### A-7 Jeder Kanzler steht im Feld der meisten
Wenn ein Gast das Ergebnis ansieht,
Dann steht jeder Kanzler in dem Feld, das die meisten gewählt haben, mit dem Anteil in Prozent und der Zahl der Stimmen.
Und bei Gleichstand steht er in jedem dieser Felder, mit dem Wort „Gleichstand“.

### A-8 Ein freiwilliger Name zeigt, wer mitgemacht hat
Wenn ein Gast beim Abstimmen seinen Namen einträgt,
Dann steht er unter dem Ergebnis bei „Abgestimmt haben: Anna, Bernd und 3 ohne Namen“.
Und niemand sieht, wer wie abgestimmt hat; den Namen kann er mit seiner Stimme ändern oder wieder löschen.
