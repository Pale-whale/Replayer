<div align="center">

# 🚀 Replayer

**Ton coach Rocket League dans le terminal.**

Parcours tes replays, analyse-les frame par frame, compare-toi aux joueurs de ton niveau et lance une session de coaching avec [Claude Code](https://claude.com/claude-code). Le coach garde une mémoire de ta progression d'une session à l'autre.

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
- [Dossier de travail](#dossier-de-travail)
- [Référence de niveau](#référence-de-niveau)
- [Sessions de coaching](#sessions-de-coaching)
- [Référence des données](#référence-des-données)
- [Précision et limites](#précision-et-limites)
- [Développement](#développement)
- [Crédits](#crédits)

## Aperçu

```text
  Analyse
                           Boost 0bst% Derr%  Déf%  Off%  1er%   SS% Touch Suiv% Rend%   50%    KO   KO+   BE!
  ★ PaleWhale                 52    11    69    25    38    52    13    32    37    43    17     4     2     3
  ★ jamb0n70                  52   9.1    75    25    36    48    20    54    50    30    18     4     3     3
    Adversaire1               50    15    70    55    10    61   9.1    57    46    35    13     3     0     2
    Adversaire2               42    19    72    59    12    39    11    26    13    35    44     4     1     2
  Réf. locale (méd., n=60)    48    14    71    45    21    50    12    35    39    38    18   3.8   0.9   2.6

  Kickoffs : bleu 5 · orange 1 · neutres 2

  But 4:55 — Adversaire1, 2 s avant
     BLEU                                ORANGE
    +-------------:------|------:-------------+
    |             :      |      :             |
    |             :      |      :             |
    |             :      B      :             |
   [|             :      |      :             |]
    |          2  :      |      :             |
    |             :      |      :             |
    |    1    o   A      |      :             |
    +-------------:------|------:-------------+
    1 PaleWhale        boost 100  hauteur   1%
    2 jamb0n70         boost  15  hauteur   5%
    A Adversaire1      boost 100  hauteur   1%
    B Adversaire2      boost   0  hauteur   1%
    o balle                       hauteur  69%
```

## Fonctionnalités

| | |
|---|---|
| 📋 **Liste des replays** | Lit l'en-tête de chaque `.replay` avec un parseur Go natif. Pour chaque match : date, victoire ou défaite et score de ton point de vue, map, mode, coéquipiers et adversaires. Filtre avec `/`. |
| 🔍 **Détail d'un match** | Scoreboard, tableau d'analyse par joueur ([colonnes](#colonnes-danalyse)), ligne de référence de ton niveau, bilan des kickoffs, et un mini-terrain ASCII 2 s avant chaque but. |
| 🧮 **Analyse des frames** | Le replay est décodé avec [`rrrocket`](https://github.com/nickbabcock/rrrocket). Replayer en tire, par joueur : positionnement, rotations, vitesse, boost, touches, possession, 50/50, démos, kickoffs. Les analyses sont mises en cache dans le [dossier de travail](#dossier-de-travail). |
| 🤝 **Possession et 50/50** | Pour chaque touche : la suivante vient-elle de toi, d'un coéquipier ou de l'adversaire ? Ou était-ce un 50/50 ? Tu vois combien de fois tu rends la balle au lieu de la garder. |
| 📊 **Référence de niveau** | Tes chiffres face aux joueurs de ton rang : par défaut les autres joueurs de tes matchs classés récents ; en option, des replays de ton rang téléchargés depuis ballchasing.com ([détails](#référence-de-niveau)). |
| 🎙️ **Coach sur un match** | `c` met la TUI en pause et ouvre une session `claude` interactive, briefée sur le match, ton niveau, ton historique et ta référence. Tu reviens dans la TUI en quittant claude. |
| 📈 **Tendances** | Coche des replays avec `espace` puis tape `t` : une ligne par match, la moyenne, la référence et l'évolution. `c` ouvre un coach sur toute la série. |
| 🧠 **Mémoire du coach** | Le coach tient un journal et une liste d'objectifs mesurables. À la session suivante, il vérifie avec ton historique si les objectifs sont atteints. |
| ✏️ **Prompt personnalisable** | Personnalité, ton, règles et structure des réponses du coach sont dans des fichiers Markdown que tu modifies avec `e`. |
| 🪪 **Alias** | Plusieurs comptes ou pseudos pour un même joueur : tout est regroupé sous son nom principal. |

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

Au démarrage, Replayer analyse en tâche de fond les replays absents du cache : environ 0,3 s par replay la première fois, puis seulement les nouveaux.

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
| `b` | construire la référence ballchasing pour la playlist du replay sélectionné | |
| `e` | éditer le [prompt du coach](#prompt-du-coach) dans `$VISUAL` / `$EDITOR` (sinon `nano` ou `vi`) | |
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
| `Suiv%` | Touches suivies d'une touche de son équipe (soi-même ou un coéquipier) |
| `Rend%` | Touches suivies d'une touche adverse : balle rendue |
| `50%` | Touches en 50/50 |
| `KO` `KO+` | Kickoffs où le joueur est allé au ballon / qu'il a gagnés |
| `BE!` | Buts encaissés alors que le joueur était devant la balle 2 s avant |

La ligne `Réf.` donne la médiane des joueurs de ton niveau ([détails](#référence-de-niveau)). Pour les compteurs (`Touch`, `KO`, `KO+`, `BE!`), elle est ramenée à 5 min de jeu.

## Configuration

Fichier `~/.config/replayer/config.json`. Les flags l'emportent sur la config.

```json
{
  "players": ["PaleWhale", "jamb0n70"],
  "aliases": { "Le pote a michel": "PaleWhale" },
  "rank": "C3",
  "ranks": { "ranked-duels": "D2" },
  "workdir": "~/Documents/replayer-data",
  "ballchasing_token": ""
}
```

| Champ | Type | Défaut | Description |
|-------|------|--------|-------------|
| `demos_dir` | string | dossier `Demos` de Steam Flatpak (Proton) | Dossier des replays |
| `players` | []string | vide | Pseudos en jeu à coacher. Le premier est le joueur principal. Sert aussi à déterminer « ton » équipe. Si la liste est vide, ou si aucun pseudo n'est dans le match, c'est le joueur qui a enregistré le replay qui est pris. |
| `aliases` | map | vide | Autres pseudos d'un joueur → son pseudo principal (second compte, changement de nom). Appliqué partout : liste, analyses, historique, coach. Changer les alias recalcule le cache d'analyse. |
| `rank` | string | vide | Ton rang (`B1`…`D3`, `C1`…`C3`, `GC1`…`GC3`, `SSL` ; `c3` et `champion-3` sont acceptés). Le rang n'est pas enregistré dans les replays : il sert au coach et à la [référence ballchasing](#référence-de-niveau). |
| `ranks` | map | vide | Rang par playlist s'il diffère de `rank`. Clés : `ranked-duels`, `ranked-doubles`, `ranked-standard`, `ranked-hoops`, `ranked-rumble`, `ranked-dropshot`, `ranked-snowday`. |
| `workdir` | string | `~/Documents/replayer-data` | [Dossier de travail](#dossier-de-travail) |
| `ballchasing_token` | string | vide | Token de l'API [ballchasing.com](https://ballchasing.com/upload). La variable d'environnement `BALLCHASING_TOKEN` est prioritaire. Ne le commite jamais. |

Le dossier `Demos` par défaut est celui de Steam installé en Flatpak (Proton) :
`~/.var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps/compatdata/252950/pfx/drive_c/users/steamuser/Documents/My Games/Rocket League/TAGame/Demos`

## Dossier de travail

```text
~/Documents/replayer-data/
├── analyses/<id>.json                 cache des analyses (recalculées si le format change)
├── benchmarks/<playlist>_<rang>/      replays ballchasing téléchargés + benchmark.json
├── memory/                            journal.md et objectifs.md, tenus par le coach
├── prompts/                           coach.md, match.md, trends.md : le prompt du coach, modifiable
└── sessions/
    ├── <id du replay>/                session coach sur un match
    └── trends-<NvN>-<joueurs>/        session coach sur une série
```

## Référence de niveau

Il y a deux sources. Dans les deux cas, les joueurs de référence passent par **la même analyse** que toi : les chiffres sont directement comparables.

| Source | Contenu | Mise en place |
|--------|---------|---------------|
| **Locale** (par défaut) | Les autres joueurs (adversaires et coéquipiers non coachés) de tes 30 derniers matchs classés de la playlist. Le matchmaking les place à ton MMR. | Rien à faire |
| **Ballchasing** | Les 40 replays les plus récents de la playlist où tous les joueurs ont ton rang configuré, téléchargés depuis ballchasing.com. | Un token gratuit dans `ballchasing_token`, puis `b` sur un replay classé |

Si une référence ballchasing existe pour la playlist et ton rang, elle remplace la locale.

Le téléchargement respecte la limite du compte gratuit, 1 fichier par seconde et 200 par heure : comptes ~1 min pour 40 replays. Un téléchargement interrompu reprend là où il s'était arrêté.

Les matchs de moins de 2 min (forfaits) et les joueurs présents moins d'une minute sont ignorés.

## Sessions de coaching

Le coach tourne dans un dossier préparé par Replayer, sous `<workdir>/sessions/`, avec un prompt système qui lui décrit tes fichiers, ton niveau déclaré et la mémoire.

`claude` est lancé avec :
- `--add-dir <workdir>/memory`, pour qu'il accède à sa mémoire ;
- `--permission-mode acceptEdits`, pour qu'il écrive son journal sans te demander. L'écriture reste limitée au dossier de session et à `memory/`.

Claude Code demande de faire confiance au dossier la première fois qu'il l'ouvre.

**Sur un match** (`sessions/<id>/`) :

| Fichier | Contenu |
|---------|---------|
| `match.json` | L'en-tête parsé (map, mode, score, scoreboard, buts) |
| `analysis.json` | L'analyse ([référence](#analysisjson)). S'il manque (`rrrocket` absent ou en échec), la TUI affiche un avertissement. |
| `history.json` | Tes 50 derniers matchs de la même playlist ([référence](#historyjson)) |
| `benchmark.json` | La référence de niveau et tes percentiles ([référence](#benchmarkjson)) |
| `replay.replay` | Lien symbolique vers le replay original |

**Sur une série** (`sessions/trends-<NvN>-<joueurs>/`) : `trends.json` ([référence](#trendsjson)), `matches/<date>_<id>.json` (`replay_file`, `match` et `analysis` de chaque match), `history.json` et `benchmark.json`. Ici, `benchmark.json` compare ta moyenne sur la série. Le nom du dossier ne dépend que du format et des joueurs : la confiance n'est demandée qu'une fois par composition.

**Mémoire** (`<workdir>/memory/`) :
- au début de chaque session, le coach lit `objectifs.md` et `journal.md` et vérifie avec `history.json` si les objectifs précédents sont atteints ;
- après sa première réponse, il ajoute une entrée datée au journal et réécrit 2 ou 3 objectifs mesurables (métrique, valeur actuelle, cible).

### Prompt du coach

Le prompt système envoyé à `claude` est assemblé ainsi :

| Partie | Origine | Contenu par défaut |
|--------|---------|--------------------|
| `prompts/coach.md` | Modifiable | Personnalité, ton et règles : s'appuyer sur les données, mini-terrain ASCII pour les positions, hauteurs en % du plafond, usage de `rrrocket`, français et tutoiement |
| Session | Générée | Ce qui est analysé, joueurs coachés, niveau déclaré, description des fichiers de la session |
| `prompts/match.md` ou `prompts/trends.md` | Modifiable | Structure de la première réponse, sur un match ou sur une série |
| Mémoire | Générée | Emplacement de `memory/` et consignes du journal et des objectifs |

Les fichiers de `prompts/` sont créés avec le texte par défaut au premier besoin, puis jamais écrasés. Modifie-les avec `e` dans la TUI ou directement. Pour revenir au texte par défaut, supprime le fichier. Les changements valent pour les sessions suivantes.

## Référence des données

Les analyses sont calculées sur le temps de jeu actif uniquement (état `Active` du match), donc hors compte à rebours et célébrations.

Unités et conventions :
- distances en uu (1 uu = 1 cm) ;
- vitesses en uu/s (max 2300, supersonique ≥ 2200) ;
- boost en % (0-100) ;
- hauteurs en % du plafond (2044 uu) ;
- `team` 0 = bleu, 1 = orange ;
- les zones sont toujours vues du joueur concerné : le tiers défensif est celui de son propre but.

### analysis.json

| Champ | Description |
|-------|-------------|
| `version` | Format du rapport (le cache est recalculé s'il change) |
| `aliases_key` | Alias appliqués lors de l'analyse (le cache est recalculé s'ils changent) |
| `playlist`, `ranked` | Playlist (nom ballchasing, ex. `ranked-doubles`), match classé ou non |
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
| `touches_in_play` | Touches hors contacts de kickoff : c'est la base des pourcentages de possession |
| `touches_followed_by_self_pct`, `touches_followed_by_teammate_pct`, `touches_kept_by_team_pct` | Touches suivies d'une touche du même joueur, d'un coéquipier, ou des deux (balle gardée) |
| `touches_given_to_opponent_pct` | Touches suivies d'une touche adverse : balle rendue |
| `fifty_fifty_pct` | Touches en 50/50 |
| `touches_unfollowed_pct` | Touches sans touche ensuite : but ou fin du match |
| `fifty_fifties`, `fifty_fifty_won`, `fifty_fifty_lost` | 50/50 disputés et leur issue (équipe qui touche ensuite, ou qui marque) |
| `demos_inflicted`, `demos_received` | Démos |
| `kickoffs`, `kickoffs_went`, `kickoffs_went_won`, `kickoffs_went_lost` | Kickoffs joués, ceux où le joueur est allé au ballon, et parmi eux gagnés / perdus |

Les cinq catégories de possession (soi-même, coéquipier, adversaire, 50/50, rien) font 100 % de `touches_in_play`.

</details>

<details>
<summary><code>goals[]</code></summary>

| Champ | Description |
|-------|-------------|
| `clock` | Horloge du match (`+m:ss (prolongation)` en prolongation) |
| `scorer`, `scoring_team` | Buteur et équipe |
| `last_touches[]` | Jusqu'à 4 touches dans les 10 s avant le but : `player`, `team`, `seconds_before_goal`, `zone` |
| `positioning_2s_before[]` | Pour chaque joueur 2 s avant le but : `map_label`, `zone`, `distance_to_ball`, `boost`, `behind_ball`, `speed`, `height_pct` |
| `ball_height_pct_2s_before` | Hauteur de la balle |
| `field_map_2s_before` | Mini-terrain ASCII vu de dessus, but bleu à gauche (`1 2 3` = bleus, `A B C` = orange, `o` = balle, `:` = limites des tiers, `\|` = milieu), suivi de sa légende |
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

### history.json

| Champ | Description |
|-------|-------------|
| `playlist` | Playlist du match (ou dominante de la série) |
| `players[].matches[]` | Les 50 derniers matchs du joueur dans cette playlist, du plus ancien au plus récent : `date`, `id`, `map`, `won`, `score`, `metrics` (les [colonnes d'analyse](#colonnes-danalyse)) |
| `players[].last10_mean`, `previous10_mean` | Moyenne des 10 derniers matchs, et des 10 d'avant |

### benchmark.json

| Champ | Description |
|-------|-------------|
| `source`, `playlist`, `rank`, `replays`, `players` | Origine et taille de la référence |
| `reference.<métrique>` | `mean`, `p25`, `median`, `p75` sur les joueurs de référence |
| `coached.<joueur>.<métrique>` | `value`, `percentile` (50 = médiane), avec les quartiles |

Toutes les métriques numériques de `players[]` sont couvertes. Les compteurs sont ramenés à 5 min de jeu. Un percentile n'est pas une note : plus de tiers défensif ou plus de 50/50 n'est pas forcément mieux.

### trends.json

| Champ | Description |
|-------|-------------|
| `notes` | Méthode, à l'intention du coach |
| `players` | Joueurs coachés présents dans la série |
| `matches[]` | Du plus ancien au plus récent : `id`, `date`, `map`, `mode`, `won`, `score_us`, `score_them`, `team_kickoffs_won` / `_lost` / `_neutral`, et `players` |
| `matches[].players.<nom>` | Tous les champs de `players[]` d'analysis.json, plus `conceded`, `conceded_not_behind_ball` et `conceded_low_boost` : les buts encaissés, et parmi eux ceux où le joueur était devant la balle ou sous 20 de boost 2 s avant |
| `trends[]` | Par joueur coaché : `matches`, `wins`, `mean` (moyenne par match des colonnes d'analyse), `evolution_recent_minus_old` (moitié récente − moitié ancienne) |

## Précision et limites

- **Rang.** Il n'est jamais enregistré dans les replays (vérifié de 2021 à 2026) : il vient de la config. La playlist, elle, est lue dans le replay.
- **Boost.** Le réseau ne l'envoie qu'au début ou à la fin d'un boost et lors des ramassages. Entre deux, il est simulé (conso de 33 %/s), donc précis à quelques % près.
  - Un ramassage est une hausse de plus de 8/255 par rapport au boost simulé ; un gros pad, une hausse de plus de 45/255.
  - Un gros pad pris presque plein compte comme un petit.
- **Touches de balle.** Elles sont détectées par heuristique : la vitesse de la balle change d'au moins 200 uu/s entre deux frames, avec une voiture à moins de 260 uu.
  - Validé sur 4 replays qui portent le compteur exact du jeu (`BallTouches`), soit 792 touches : rappel de 90 %, précision de 96 %.
  - La dernière touche détectée avant un but correspond au buteur annoncé par le jeu.
- **50/50.** Les deux équipes touchent la balle à moins de 0,3 s d'intervalle, ou un adversaire est à moins de 200 uu de la balle lors de la touche.
  - Validé sur les mêmes 4 replays (68 duels) : rappel de 90 %, précision de 92 %.
  - Les contacts de kickoff sont exclus : ils sont comptés dans les kickoffs.
- **Kickoffs.** Sur 105 replays de test (734 kickoffs), le nombre de kickoffs correspond au nombre de buts dans tous les matchs. 97 % des `time_to_ball` sont entre 1,8 et 4 s.
- **Hauteur.** La hauteur seule ne distingue pas un joueur en l'air d'un joueur qui roule au mur. En hoops, le plafond est plus bas que 2044 uu, donc les % sont approximatifs.
- **Référence.**
  - Locale : elle dépend de ton MMR au moment des matchs.
  - Ballchasing : l'échantillon ne compte que des joueurs qui publient leurs replays. Au-dessus de C3, l'API ne distingue pas GC1, GC2, GC3 et SSL.

## Développement

```sh
go vet ./...
go test ./...                                  # tests unitaires
RL_DEMOS=/chemin/vers/Demos go test ./... -v   # + tests sur de vrais replays
```

Avec `RL_DEMOS`, les tests :
- parsent et analysent tous les replays du dossier ;
- mesurent les heuristiques de touches et de 50/50 contre le compteur exact du jeu ;
- valident les kickoffs, la possession, la référence locale et les tendances.

Le client ballchasing est testé contre un faux serveur (`httptest`).

| Package | Rôle |
|---------|------|
| `internal/replay` | Parseur natif de l'en-tête `.replay`, scan du dossier, point de vue des joueurs coachés |
| `internal/analysis` | Décodage via `rrrocket -n`, timeline des frames, métriques, kickoffs, possession, mini-terrains |
| `internal/rank` | Rangs et playlists (noms ballchasing) |
| `internal/store` | Dossier de travail : cache des analyses, sessions, mémoire, références |
| `internal/benchmark` | Référence de niveau locale et ballchasing, percentiles |
| `internal/trends` | Agrégation de plusieurs analyses, historique, colonnes affichées par la TUI |
| `internal/coach` | Préparation des dossiers de session, prompt du coach (fichiers modifiables) et lancement de `claude` |
| `internal/config` | Lecture de `~/.config/replayer/config.json` |
| `internal/tui` | Interface Bubble Tea : liste, détail, tendances |

## Crédits

- [rrrocket](https://github.com/nickbabcock/rrrocket) / [boxcars](https://github.com/nickbabcock/boxcars) de Nick Babcock, pour le décodage des replays
- [ballchasing.com](https://ballchasing.com), pour les replays de référence
- [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles) et [Lip Gloss](https://github.com/charmbracelet/lipgloss) de Charm
- [Claude Code](https://claude.com/claude-code) d'Anthropic, pour le coach

*Rocket League est une marque de Psyonix. Ce projet n'est ni affilié à Psyonix ni soutenu par eux.*
