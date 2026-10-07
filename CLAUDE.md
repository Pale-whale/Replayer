# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

## Documentation
Write in a README.md:
- all the features of the operator and how to use them
- all the CRD reference

The README.md and CLAUDE.md *MUST* be keep in sync with the code, if a feature is added or modified it must be reflected in the .md

## Projet

TUI Go (Bubble Tea) qui liste les replays Rocket League et lance une session `claude` interactive de coaching sur un replay.

- `main.go` : flags (`-dir`, `-players`), chargement config, lancement de la TUI.
- `internal/config` : `~/.config/replayer/config.json` (`demos_dir`, `players`).
- `internal/replay` : parseur natif de l'en-tête `.replay` (propriétés Unreal : la taille d'une propriété est un i32 suivi d'un i32 d'index de tableau ; un `ByteProperty` dont le kind vaut `"None"` porte un octet brut). `Scan` lit un dossier, du plus récent au plus ancien.
- `internal/analysis` : `File(r)` lance `rrrocket -n` puis calcule le rapport `analysis.json`.
  - `timeline.go` : rejoue les frames réseau → snapshots (balle + voitures) aux frames où l'état du match est `Active`, ramassages de boost, démos.
    - Les keyframes (toutes les ~10 s) renvoient `new_actors` avec les mêmes ids et ré-émettent les compteurs : on suit les acteurs par id et on ne compte que les incréments.
    - Le boost n'est répliqué qu'aux changements : on simule la conso (255/3 par seconde).
    - La voiture démolie perd son `PlayerReplicationInfo` juste avant l'attribut de démo : on garde le dernier propriétaire.
    - Les démos ont 3 formats : `Demolish`, `DemolishFx` (champs plats) et `DemolishExtended`.
    - Une même frame peut supprimer puis recréer un id : on retire l'id d'`actorObj` à la suppression.
    - Un RigidBody sans `linear_velocity` est au repos : vitesse nulle.
    - `KickoffStarts` = 1er snapshot `Active` après un `Countdown`. En prolongation, `SecondsRemaining` compte le temps écoulé.
  - `metrics.go` : métriques par joueur, détection heuristique des touches, contexte des buts.
  - `kickoff.go` : kickoffs. La 1re touche est le moment où la vitesse horizontale de la balle dépasse 100 uu/s (en Hoops, la balle est lancée verticalement). On en tire qui y va, le gagnant et la position de départ.
- `internal/replay/players.go` : `Coached`, `OurTeam`, `Scores` (point de vue des joueurs coachés).
- `internal/trends` : `Build` agrège plusieurs rapports (par match + moyenne + évolution récents − anciens). `Metrics` définit les colonnes affichées par la TUI (détail et tendances). `ForPlayer` ajoute les stats de buts encaissés.
- `internal/coach` :
  - `Command(r, rep, players)` prépare `~/.cache/replayer/sessions/<id>/` (`match.json`, `analysis.json` si `rep != nil`, lien `replay.replay`).
  - `TrendsCommand` prépare `sessions/trends-<NvN>-<joueurs>/` (`trends.json`, `matches/*.json`).
  - Les deux lancent `claude --append-system-prompt <coach> "<premier message>"`.
- `internal/tui` :
  - `tui.go` : modèle, touches, pages liste / détail / tendances.
    - Les analyses sont calculées en tâche de fond (`analyze`, 4 workers) et mises en cache dans `Model.reports`.
    - `reportsMsg.then` enchaîne l'action : rafraîchir, lancer le coach, ouvrir les tendances.
    - Le coach est lancé via `tea.ExecProcess`, ce qui suspend la TUI.
  - `views.go` : rendu des pages détail et tendances, dans un viewport.

Tests sur de vrais replays : `RL_DEMOS=<dossier Demos> go test ./... -v`.
- Ils parsent et analysent tous les replays.
- `TestTouchHeuristic` mesure l'heuristique des touches contre le compteur `BallTouches`, présent seulement dans les replays récents.
- `TestKickoffs` vérifie le nombre de kickoffs par match et la plausibilité des résultats.
- `internal/trends` teste les tendances sur les 10 derniers 2v2 PaleWhale + jamb0n70.
