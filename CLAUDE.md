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

TUI Go (Bubble Tea) qui liste les replays Rocket League, les analyse frame par frame, les compare à une référence de niveau et lance des sessions `claude` interactives de coaching, avec une mémoire entre les sessions.

- `main.go` : flags (`-dir`, `-players`), chargement config, lancement de la TUI.
- `internal/config` : `~/.config/replayer/config.json`.
  - Champs : `demos_dir`, `players`, `aliases` (pseudo secondaire → pseudo principal), `workdir` (`~/` développé), `rank` / `ranks` (normalisés via `rank.Normalize`), `ballchasing_token`.
  - La variable d'environnement `BALLCHASING_TOKEN` est prioritaire sur le token de la config.
  - `RankFor(playlist)` donne le rang d'une playlist.
- `internal/rank` : normalisation des rangs (`C3`), valeurs ballchasing, playlist id → nom, `Ranked`.
  - Le rang n'est JAMAIS dans les replays : `PRI_TA:SkillTier` n'est pas répliqué. La playlist l'est, via `ReplicatedGamePlaylist`.
- `internal/replay` : parseur natif de l'en-tête `.replay` (propriétés Unreal : la taille d'une propriété est un i32 suivi d'un i32 d'index de tableau ; un `ByteProperty` dont le kind vaut `"None"` porte un octet brut). `Scan` lit un dossier, du plus récent au plus ancien.
  - `players.go` : `ApplyAliases` (appelé dans `main` après `Scan`), `Coached`, `OurTeam`, `Scores` (point de vue des joueurs coachés). `Coached` se rabat sur le joueur qui a enregistré le replay si aucun joueur configuré n'est présent.
- `internal/analysis` : `File(r, aliases)` lance `rrrocket -n` puis calcule le rapport `analysis.json`.
  - Les alias sont appliqués aux noms des PRI dans `buildTimeline`, via `netReplay.aliases`.
  - `Report.AliasesKey` identifie les alias appliqués.
  - Incrémenter `Version` quand le format du rapport change : le cache est alors recalculé.
  - `timeline.go` : rejoue les frames réseau → snapshots (balle + voitures) aux frames où l'état du match est `Active`, ramassages de boost, démos, playlist.
    - Les keyframes (toutes les ~10 s) renvoient `new_actors` avec les mêmes ids et ré-émettent les compteurs : on suit les acteurs par id et on ne compte que les incréments.
    - Le boost n'est répliqué qu'aux changements : on simule la conso (255/3 par seconde).
    - La voiture démolie perd son `PlayerReplicationInfo` juste avant l'attribut de démo : on garde le dernier propriétaire.
    - Les démos ont 3 formats : `Demolish`, `DemolishFx` (champs plats) et `DemolishExtended`.
    - Une même frame peut supprimer puis recréer un id : on retire l'id d'`actorObj` à la suppression.
    - Un RigidBody sans `linear_velocity` est au repos : vitesse nulle.
    - `KickoffStarts` = 1er snapshot `Active` après un `Countdown`. En prolongation, `SecondsRemaining` compte le temps écoulé.
  - `metrics.go` : métriques par joueur, détection heuristique des touches (`Contested` = un adversaire à moins de `fiftyContactDist`), contexte des buts (hauteurs en %, mini-terrain).
  - `kickoff.go` : kickoffs. La 1re touche est le moment où la vitesse horizontale de la balle dépasse 100 uu/s (en Hoops, la balle est lancée verticalement). On en tire qui y va, le gagnant et la position de départ.
  - `possession.go` : touches hors kickoff regroupées en clusters (écart ≤ 0,3 s) ; un cluster est un 50/50 s'il contient les deux équipes ou un contact adverse. Chaque autre touche est classée selon la suivante : soi, coéquipier, adversaire, rien. Les seuils sont calés sur le compteur `BallTouches`.
  - `fieldmap.go` : mini-terrain ASCII 41×7 vu de dessus (but bleu à gauche), légende avec boost et hauteur en % du plafond (2044 uu).
- `internal/store` : dossier de travail (`analyses/`, `benchmarks/`, `memory/`, `prompts/`, `sessions/`). `Analysis(r)` passe par le cache. Un rapport n'est réutilisé que si `Version` et `AliasesKey` correspondent.
- `internal/benchmark` : référence de niveau.
  - `Local` : autres joueurs des 30 derniers matchs classés de la playlist.
  - `Ballchasing.Build` : liste + téléchargement à 1/s max (reprise possible), analyse, `benchmark.json`.
  - `Values` : toutes les métriques numériques par réflexion, compteurs ramenés à 5 min de jeu.
  - `Compare` : percentiles.
- `internal/trends` :
  - `Build` agrège plusieurs rapports (par match + moyenne + évolution récents − anciens).
  - `BuildHistory` donne l'historique par joueur configuré pour une playlist.
  - `Metrics` définit les colonnes affichées par la TUI (détail et tendances).
  - `ForPlayer` ajoute les stats de buts encaissés.
- `internal/coach` : `Env` (store, joueurs, rang, toutes les analyses, référence).
  - `Command` prépare `<workdir>/sessions/<id>/` (`match.json`, `analysis.json`, `history.json`, `benchmark.json`, lien `replay.replay`).
  - `TrendsCommand` prépare `sessions/trends-<NvN>-<joueurs>/` (`trends.json`, `matches/*.json`, `history.json`, `benchmark.json`).
  - Les deux lancent `claude --add-dir <workdir>/memory --permission-mode acceptEdits --append-system-prompt <prompt> "<premier message>"`.
  - Le prompt est assemblé ainsi :
    1. `prompts/coach.md` (modifiable) ;
    2. la session (générée par `session`) ;
    3. `prompts/match.md` ou `trends.md` (modifiable) ;
    4. la mémoire (générée), qui demande de tenir `memory/journal.md` et `memory/objectifs.md`.
  - `prompts.go` : textes par défaut, `PromptFiles` crée les fichiers manquants sans jamais écraser les existants.
- `internal/tui` :
  - `tui.go` : modèle, touches, pages liste / détail / tendances.
    - `Init` analyse tous les replays en tâche de fond (cache).
    - `analyze` utilise 4 workers ; `reportsMsg.then` enchaîne l'action.
    - `bench(playlist)` est mis en cache et vidé quand les analyses changent.
    - `b` construit la référence ballchasing ; la progression arrive par un canal (`benchProgressMsg`).
    - `e` ouvre les fichiers de prompt dans `$VISUAL` / `$EDITOR` (sinon `nano` ou `vi`).
    - Le coach est lancé via `tea.ExecProcess`.
  - `views.go` : rendu des pages détail (tableau + ligne de référence + mini-terrains colorés des buts) et tendances, dans un viewport.

Tests sur de vrais replays : `RL_DEMOS=<dossier Demos> go test ./... -v`.
- Ils parsent et analysent tous les replays.
- `TestTouchHeuristic` et `TestFiftyFifty` mesurent les heuristiques contre le compteur `BallTouches`, présent seulement dans les replays récents.
- `TestKickoffs` vérifie le nombre de kickoffs par match.
- `TestPossessionSums` vérifie que les catégories de possession font 100 %.
- `TestLocal` (benchmark) et `internal/trends` testent la référence locale et les tendances sur les 2v2 PaleWhale + jamb0n70.
- `TestBallchasingBuild` utilise un faux serveur `httptest` ; avec `RL_DEMOS`, il analyse aussi un vrai replay.
- `TestCommandUsesEditedPrompts` vérifie l'assemblage du prompt et que les fichiers existants ne sont pas écrasés.
