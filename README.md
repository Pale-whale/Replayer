<div align="center">

# 🚀 Replayer

**Ton coach Rocket League dans le terminal.**

Parcours tes replays, analyse-les frame par frame et lance une session de coaching avec [Claude Code](https://claude.com/claude-code) sur un match ou sur une série de matchs.

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)
![TUI](https://img.shields.io/badge/TUI-Bubble%20Tea-FF5F87)
![Claude Code](https://img.shields.io/badge/coach-Claude%20Code-D97757)
![Platform](https://img.shields.io/badge/platform-Linux-informational?logo=linux&logoColor=white)

</div>

---

## Sommaire

- [Aperçu](#aperçu)
- [Fonctionnalités](#fonctionnalités)
- [Installation](#installation)
- [Utilisation](#utilisation)
- [Configuration](#configuration)
- [Sessions de coaching](#sessions-de-coaching)
- [Référence des données](#référence-des-données)
- [Précision et limites](#précision-et-limites)
- [Développement](#développement)
- [Crédits](#crédits)

## Aperçu

```text
  cs_day_p  Défaite 3-4
  06/10/2026 22:18 · Online 2v2 · E31BC47211F1C1D34A3BDABA21291C79

  Bleu   3
                            Score  Buts    PD  Arrêts  Tirs
  ★ PaleWhale                 275     1     0       1     1
  ★ jamb0n70                  512     2     1       0     8

  Orange 4
                            Score  Buts    PD  Arrêts  Tirs
    Adversaire1               758     3     1       1     4
    Adversaire2               534     1     2       3     2

  Analyse
                            Boost  0bst%  Derr%   Déf%   Off%   1er%    SS%  Touch     KO    KO+    BE!
  ★ PaleWhale                  52     11     69     25     38     52     13     32      4      2      3
  ★ jamb0n70                   52    9.1     75     25     36     48     20     54      4      3      3
    Adversaire1                50     15     70     55     10     61    9.1     57      3      0      2
    Adversaire2                42     19     72     59     12     39     11     26      4      1      2

  Kickoffs : bleu 5 · orange 1 · neutres 2

  ↑/↓ défiler · c coach · esc retour · ctrl+c quitter
```

## Fonctionnalités

| | |
|---|---|
| 📋 **Liste des replays** | Lit l'en-tête de chaque `.replay` avec un parseur Go natif, sans dépendance. Pour chaque match : date, victoire ou défaite et score de ton point de vue, map, mode, coéquipiers et adversaires. Filtre avec `/`. |
| 🔍 **Détail d'un match** | Scoreboard, tableau d'analyse par joueur ([colonnes](#colonnes-danalyse)), bilan des kickoffs, chronologie des buts. Tes coéquipiers et toi êtes marqués d'une `★`. |
| 🧮 **Analyse des frames** | Le replay est décodé avec [`rrrocket`](https://github.com/nickbabcock/rrrocket). Replayer en tire, par joueur : positionnement, rotations, vitesse, boost, touches, démos. Il ajoute le contexte de chaque but (qui était où, avec combien de boost) et le détail de chaque kickoff (qui y va, qui gagne). |
| 🎙️ **Coach sur un match** | `c` met la TUI en pause et ouvre une session `claude` interactive, déjà briefée sur le match. Tu reviens dans la TUI en quittant claude. |
| 📈 **Tendances** | Coche des replays avec `espace` puis tape `t`. Pour chaque joueur coaché : une ligne par match, la moyenne, et l'évolution entre les matchs récents et les anciens. `c` ouvre un coach sur toute la série : problèmes récurrents, progrès, plan d'entraînement. |

## Installation

### Prérequis

| Outil | Rôle |
|-------|------|
| [Go](https://go.dev/dl/) 1.26+ | Compiler Replayer |
| [Claude Code](https://claude.com/claude-code) | Le coach (commande `claude` dans le `PATH`) |
| [`rrrocket`](https://github.com/nickbabcock/rrrocket) | Décoder les frames réseau des replays (dans le `PATH`) |

`rrrocket` n'est pas publié sur crates.io. Prends le binaire des [releases GitHub](https://github.com/nickbabcock/rrrocket/releases) :

```sh
curl -sL https://github.com/nickbabcock/rrrocket/releases/download/v0.11.6/rrrocket-0.11.6-x86_64-unknown-linux-musl.tar.gz | tar xz
install -m755 rrrocket-0.11.6-x86_64-unknown-linux-musl/rrrocket ~/.local/bin/
```

Sans `rrrocket`, la liste et le scoreboard fonctionnent, mais le coach n'a que les stats de fin de match.

### Compilation

```sh
git clone git@github.com:Pale-whale/Replayer.git
cd Replayer
go build -o replayer .
```

## Utilisation

```sh
./replayer
./replayer -players "PaleWhale,jamb0n70" -dir /chemin/vers/Demos
```

### Flags

| Flag | Défaut | Description |
|------|--------|-------------|
| `-dir` | `demos_dir` de la config | Dossier des replays |
| `-players` | `players` de la config | Pseudos à coacher, séparés par des virgules (le premier = toi) |

### Touches

| Touche | Liste | Détail / Tendances |
|--------|-------|--------------------|
| `↑` `↓` | naviguer | défiler (aussi `pgup` / `pgdown`) |
| `/` | filtrer | |
| `entrée` | ouvrir le détail | |
| `espace` | cocher / décocher le replay | page suivante |
| `x` | tout décocher | |
| `t` | tendances des replays cochés (au moins 2) | |
| `c` | coach sur le match sélectionné | coach sur le match / sur la série |
| `esc` `q` | quitter (`q`) | retour à la liste |
| `ctrl+c` | quitter | quitter |

### Colonnes d'analyse

Ce sont les mêmes colonnes dans la vue détail (une ligne par joueur) et dans la vue tendances (une ligne par match) :

| Colonne | Description |
|---------|-------------|
| `Boost` | Boost moyen (%) |
| `0bst%` | Temps à 0 boost |
| `Derr%` | Temps derrière la balle |
| `Déf%` `Off%` | Temps dans son tiers défensif / offensif |
| `1er%` | Temps en 1er homme (le plus proche de la balle de son équipe) |
| `SS%` | Temps en supersonique |
| `Touch` | Touches de balle |
| `KO` `KO+` | Kickoffs où le joueur est allé au ballon / qu'il a gagnés |
| `BE!` | Buts encaissés alors que le joueur était devant la balle 2 s avant |

## Configuration

Fichier optionnel `~/.config/replayer/config.json`. Les flags l'emportent sur la config.

```json
{
  "demos_dir": "/home/moi/Documents/My Games/Rocket League/TAGame/Demos",
  "players": ["PaleWhale", "jamb0n70"]
}
```

| Champ | Type | Défaut | Description |
|-------|------|--------|-------------|
| `demos_dir` | string | dossier `Demos` de Steam Flatpak (Proton) | Dossier des replays |
| `players` | []string | vide | Pseudos en jeu à coacher. Le premier est le joueur principal. Sert aussi à déterminer « ton » équipe (victoire ou défaite). Si la liste est vide, ou si aucun pseudo n'est dans le match, c'est le joueur qui a enregistré le replay qui est pris. |

Le dossier par défaut est celui de Steam installé en Flatpak (Proton) :
`~/.var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps/compatdata/252950/pfx/drive_c/users/steamuser/Documents/My Games/Rocket League/TAGame/Demos`

## Sessions de coaching

Le coach tourne dans un dossier préparé par Replayer, sous `~/.cache/replayer/sessions/`. Il démarre avec un prompt système de coach, en français et centré sur les joueurs coachés, qui décrit les fichiers à sa disposition. Il peut aussi relancer `rrrocket -n` pour creuser un point précis. Claude Code demande de faire confiance au dossier la première fois qu'il l'ouvre.

### Sur un match

`sessions/<id du replay>/` :

- `match.json` : l'en-tête parsé (map, mode, score, scoreboard, buts avec frame et seconde) ;
- `analysis.json` : les métriques calculées depuis les frames ([référence](#analysisjson)). S'il manque (`rrrocket` absent ou en échec), la TUI affiche un avertissement ;
- `replay.replay` : un lien symbolique vers le replay original.

Premier message envoyé au coach : « Analyse ce match et coache-moi. »

### Sur une série (tendances)

`sessions/trends-<NvN>-<joueurs>/` :

- `trends.json` : les tendances ([référence](#trendsjson)) ;
- `matches/<date>_<id>.json` : `replay_file`, `match` (en-tête) et `analysis` (analysis.json complet) de chaque match.

Le nom du dossier ne dépend que du format et des joueurs coachés : Claude Code ne demande la confiance qu'une fois par composition.

## Référence des données

Les analyses sont calculées sur le temps de jeu actif uniquement (état `Active` du match), donc hors compte à rebours et célébrations.

Unités et conventions :
- distances en uu (1 uu = 1 cm) ;
- vitesses en uu/s (max 2300, supersonique ≥ 2200) ;
- boost en % (0-100) ;
- `team` 0 = bleu, 1 = orange ;
- les zones sont toujours vues du joueur concerné : le tiers défensif est celui de son propre but.

### analysis.json

| Champ | Description |
|-------|-------------|
| `notes` | Méthode et limites, à l'intention du coach |
| `in_play_seconds` | Durée de jeu actif |
| `players[]` | Métriques par joueur (tous les joueurs du match) |
| `goals[]` | Contexte de chaque but |
| `kickoffs[]` | Détail de chaque kickoff |
| `demos[]` | `clock`, `attacker`, `victim` |

<details>
<summary><code>players[]</code></summary>

| Champ | Description |
|-------|-------------|
| `name`, `team`, `in_play_seconds` | Identité et temps de jeu |
| `defensive_third_pct`, `middle_third_pct`, `offensive_third_pct` | Temps passé dans chaque tiers du terrain |
| `behind_ball_pct` | Temps entre la balle et son propre but |
| `avg_distance_to_ball` | Distance moyenne à la balle |
| `first_man_pct`, `last_man_pct` | Temps comme coéquipier le plus proche de la balle / le plus proche de son but (seulement avec des coéquipiers) |
| `avg_speed`, `supersonic_pct` | Vitesse moyenne, temps en supersonique |
| `ground_pct`, `low_air_or_wall_pct`, `high_air_or_wall_pct` | Hauteur : au sol (< 40 uu), bas (< 300 uu), haut. Ne distingue pas l'air du mur |
| `avg_boost`, `zero_boost_pct`, `under_25_boost_pct`, `over_80_boost_pct` | Boost moyen et temps passé à 0, sous 25 %, au-dessus de 80 % |
| `big_pads`, `small_pads`, `avg_boost_before_big_pad` | Pads ramassés, boost moyen au moment de prendre un gros pad |
| `touches`, `touches_defensive_third`, `touches_middle_third`, `touches_offensive_third` | Touches de balle, par tiers |
| `demos_inflicted`, `demos_received` | Démos |
| `kickoffs`, `kickoffs_went`, `kickoffs_went_won`, `kickoffs_went_lost` | Kickoffs joués, ceux où le joueur est allé au ballon, et parmi eux gagnés / perdus |

</details>

<details>
<summary><code>goals[]</code></summary>

| Champ | Description |
|-------|-------------|
| `clock` | Horloge du match (`+m:ss (prolongation)` en prolongation) |
| `scorer`, `scoring_team` | Buteur et équipe |
| `last_touches[]` | Jusqu'à 4 touches dans les 10 s avant le but : `player`, `team`, `seconds_before_goal`, `zone` |
| `positioning_2s_before[]` | Pour chaque joueur 2 s avant le but : `zone`, `distance_to_ball`, `boost`, `behind_ball`, `speed` |
| `demolished_or_respawning_2s_before` | Joueurs absents du terrain à ce moment |

</details>

<details>
<summary><code>kickoffs[]</code></summary>

| Champ | Description |
|-------|-------------|
| `clock` | Horloge au coup d'envoi (en prolongation : temps écoulé depuis son début) |
| `winner` | `bleu`, `orange` ou `neutre`. C'est l'équipe qui marque dans les 3 s après la 1re touche. Sinon, c'est la moitié de terrain où se trouve la balle 3 s après (`neutre` à moins de 1000 uu du milieu) : balle chez les orange → `bleu` gagne. |
| `time_to_ball` | Secondes entre le top départ et la 1re touche, c'est-à-dire quand la balle quitte le centre |
| `goal_within_10s` | Équipe qui marque dans les 10 s après la 1re touche (absent sinon) |
| `players[]` | `player`, `team`, `spawn` (`diagonale`, `décalé` ou `fond`), `went` : le joueur de son équipe le plus proche de la balle à la 1re touche. Personne n'y est allé si ce joueur est à plus de 800 uu (fake). |

</details>

### trends.json

| Champ | Description |
|-------|-------------|
| `notes` | Méthode, à l'intention du coach |
| `players` | Joueurs coachés présents dans la série |
| `matches[]` | Du plus ancien au plus récent : `id`, `date`, `map`, `mode`, `won`, `score_us`, `score_them`, `team_kickoffs_won` / `_lost` / `_neutral`, et `players` |
| `matches[].players.<nom>` | Tous les champs de `players[]` d'analysis.json, plus `conceded`, `conceded_not_behind_ball` et `conceded_low_boost` : les buts encaissés, et parmi eux ceux où le joueur était devant la balle ou sous 20 de boost 2 s avant |
| `trends[]` | Par joueur coaché : `matches`, `wins`, `mean` (moyenne par match des colonnes d'analyse), `evolution_recent_minus_old` (moitié récente − moitié ancienne) |

## Précision et limites

- **Boost.** Le réseau ne l'envoie qu'au début ou à la fin d'un boost et lors des ramassages. Entre deux, il est simulé (conso de 33 %/s), donc précis à quelques % près.
  - Un ramassage est une hausse de plus de 8/255 par rapport au boost simulé ; un gros pad, une hausse de plus de 45/255.
  - Un gros pad pris presque plein compte comme un petit.
- **Touches de balle.** Elles sont détectées par heuristique : la vitesse de la balle change d'au moins 200 uu/s entre deux frames, avec une voiture à moins de 260 uu.
  - Sur le seul replay de test qui porte le compteur exact du jeu (`BallTouches`), elle retrouve 93 % des touches, avec 95 % de détections justes.
  - La dernière touche détectée avant un but correspond au buteur annoncé par le jeu.
- **Kickoffs.** Sur 103 replays de test (718 kickoffs), le nombre de kickoffs correspond au nombre de buts dans tous les matchs. 97 % des `time_to_ball` sont entre 1,8 et 4 s.
- **Hauteur.** La hauteur seule ne distingue pas un joueur en l'air d'un joueur qui roule au mur.

## Développement

```sh
go vet ./...
go test ./...                       # tests unitaires
RL_DEMOS=/chemin/vers/Demos go test ./... -v   # + tests sur de vrais replays
```

Avec `RL_DEMOS`, les tests parsent et analysent tous les replays du dossier, mesurent l'heuristique de touches et valident les kickoffs et les tendances.

| Package | Rôle |
|---------|------|
| `internal/replay` | Parseur natif de l'en-tête `.replay`, scan du dossier, point de vue des joueurs coachés |
| `internal/analysis` | Décodage via `rrrocket -n`, timeline des frames, métriques, kickoffs |
| `internal/trends` | Agrégation de plusieurs analyses, colonnes affichées par la TUI |
| `internal/coach` | Préparation des dossiers de session et lancement de `claude` |
| `internal/config` | Lecture de `~/.config/replayer/config.json` |
| `internal/tui` | Interface Bubble Tea : liste, détail, tendances |

## Crédits

- [rrrocket](https://github.com/nickbabcock/rrrocket) / [boxcars](https://github.com/nickbabcock/boxcars) de Nick Babcock, pour le décodage des replays
- [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) et [Lip Gloss](https://github.com/charmbracelet/lipgloss) de Charm
- [Claude Code](https://claude.com/claude-code) d'Anthropic, pour le coach

*Rocket League est une marque de Psyonix. Ce projet n'est ni affilié à Psyonix ni soutenu par eux.*
