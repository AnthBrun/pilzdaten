# Aufgabe: Pilzdatabank-Webapp aus existierenden Daten

## Kontext
Ein kompletter Crawl der Pilzdatabank 123pilzsuche.de liegt lokal:
- Strukturierte Daten: `/home/hermes/projects/pilzsuche/data/species.jsonl`
  - 4.200 Zeilen (4.179 echte Arten + 21 Meta/Listen-Seiten), Schema:
    `{slug, url, name_trivial, name_wissenschaftlich, synonyme[], merkmale{...}, bilddateien[], verwechslungen[], links_extern{}}`
    - `merkmale` hat ~20 variable Felder (geruch, geschmack, hut, fleisch, stiel, lamellen, sporenpulverfarbe, vorkommen, gattung, verwechslungsgefahr, kommentar, wiki-link, priorität, fruchtkörper, chemische reaktionen, bemerkung, besonderheit, relativer speisewert, vergleich, ring)
    - `bilddateien` = Pfade relativ zu `/home/hermes/projects/pilzsuche/raw/images/` (z. B. `daten/details/BittererKrempentrichterling-Dateien/image002.gif`)
- Indizes: `/home/hermes/projects/pilzsuche/data/index_arts.jsonl` (alphabetisch), `index_gattungen.json`, `index_verwechslungen.json`
- Bilder: 95.431 Dateien (~900 MB) unter `/home/hermes/projects/pilzsuche/raw/images/`

## Ziel
Eine **leichtgewichtige, moderne Web-App** die diese Daten als Pilzdatenbank anbietet, deployed via **Docker (Compose)**. Arbeitsverzeichnis: `/home/hermes/projects/pilzdaten` (Git-Repo, bereits init).

## Anforderungen
1. **Features:**
   - Startseite mit Suche (Live-Suche nach Trivialname, wiss. Name, Synonym — case-insensitiv)
   - Alphabet-Übersicht (A–Z, Paginierung oder Lazy)
   - Artenseite: Name, Synonyme, alle Merkmale schön gerendert, alle Bilder (lazy loading), Verwechslungspartner als klickbare Links
   - Gattungs-Übersicht (optional, wenn günstig umsetzbar)
2. **Tech-Stack — deine Wahl, aber mit diesen Grenzen:**
   - Modern und 2026-tauglich (kein jQuery/PHP-2005-Vibe)
   - **Leichtgewichtig, kein Enterprise-Ballast**: keine Microservices, kein Kubernetes, kein ORM-Schwergewicht, kein Framework mit 500 MB Dependency-Baum
   - Docker-Compose mit wenig Containern (idealerweise 1–2)
   - Die Daten sind bereits strukturiert (JSONL) — eine Datenbank-Instanz mit Migration ist NICHT nötig, außer du findest einen echten Grund (z. B. Volltextsuche). Wenn du eine DB nutzt, muss der Import ein einfaches Skript sein.
3. **Docker:**
   - `docker-compose.yml` + Dockerfile(s), Multi-Stage Build, kleiner Image
   - Bilder (900 MB) werden als **volumes eingebunden, NICHT ins Image gebaut**
   - App muss nach `docker compose up -d` direkt erreichbar sein (Port 8080)
4. **Qualität:**
   - Sauberer Code, kleine Module, README mit Setup- und Betriebs-Anleitung
   - `curl`-Smoke-Test im README (Such-Endpoint, eine Artenseite)
   - Keine TODOs-Grab: alles gefordert ist, muss laufen

## WICHTIG — Arbeitsweise
- Lies NUR diese Dateien: `data/species.jsonl` (erste 5 Zeilen fürs Schema), `data/index_arts.jsonl` (erste 5 Zeilen), `data/STATISTIK.json`. Nicht die gesamte 1.6-GB-Rawdaten.
- Leg dich VOR dem Coden auf genau EINEN Stack fest und begründe in 2 Sätzen im README (was und warum leichtgewichtig).
- Bau in Etappen: 1) Stack-Scaffold + Datenimport-Logik, 2) Suche + Liste, 3) Detailseiten + Bilder, 4) Docker + Smoke-Test.
- Docker-Tests mit `sg docker -c "docker compose ..."` (User ist in Gruppe docker, braucht `sg docker`-Wrapper).
- Git: nach jeder Etappe commiten.
- Am Ende: `docker compose up -d` muss laufen, 3 curl-Smoke-Tests grün, dann abschließender Commit.

## Abnahme
- `sg docker -c "cd /home/hermes/projects/pilzdaten && docker compose up -d"` → App auf Port 8080
- Suche auf "Krempen" liefert den Bitteren Krempentrichterling
- Artenseite zeigt Merkmale + Bilder (mindestens 1 Bild lädt)
- Image-Gesamtbau < 500 MB (Bilder kommen per Volume)
