# tmssh

Sélecteur de connexions SSH pour tmux — fzf, tags, multi-connexion, historique.
Zéro dépendance Go (stdlib uniquement) ; requiert `fzf` et `tmux` sur la machine.

## Installation

```sh
go build -o tmssh .
mv tmssh ~/.local/bin/   # ou n'importe où dans $PATH
```

## Binding tmux (ctrl-b ctrl-s)

Dans `~/.tmux.conf` :

```tmux
bind-key C-s display-popup -E -w 80% -h 60% "tmssh"
```

Puis `tmux source-file ~/.tmux.conf`.

## Utilisation

| Touche | Action |
|---|---|
| `Tab` | multi-sélection de hosts |
| `ctrl-t` | sélectionner **tous** les hosts affichés (après recherche/filtre) |
| `Enter` | connexion (1 host → window, N hosts → panes tiled) |
| `ctrl-a` | ajouter un tag aux hosts sélectionnés |
| `ctrl-d` | retirer un tag commun aux hosts sélectionnés |
| `ctrl-f` | tag picker : filtrer la liste par tags (sémantique ET) |
| `ctrl-g` | effacer le filtre par tags |
| `ctrl-s` | sauvegarder la sélection comme groupe de connexion (prompt du nom) |
| `ctrl-e` | picker de groupes : `Enter` connecte, multi-sélection = union, `ctrl-d` supprime |
| `ctrl-h` | preview enrichi du host sous le curseur (latence, tags, historique, backups) |

Workflow typique « connecte-moi à tout le prod web » :
`ctrl-f` → sélectionner `prod` + `web` → `Enter` → `ctrl-t` → `Enter`.

## Personnalisation

- **Icônes** : éditables dans `hosts.json` (`icons` associe tag → icône
  Nerd Font, `default_icon` pour les hosts sans tag iconifié).
- **Couleurs des tags** : palette fixe pour prod/staging/dev/web/db,
  couleur stable dérivée du nom pour les autres (`internal/ui/ui.go`).
- **Thème fzf** : variable `fzfTheme` dans `internal/ui/ui.go`
  (palette tokyonight par défaut).

`TMSSH_SYNC=1 tmssh` active `synchronize-panes` en multi-connexion.

## Groupes de connexion

Un groupe est l'union de hosts explicites et d'une requête de tags
(sémantique ET), stocké dans `hosts.json` :

```json
"groups": {
  "prod-web": { "hosts": ["bastion"], "tags": ["prod", "web"] },
  "all-db":   { "tags": ["db"] }
}
```

- `ctrl-s` dans l'UI sauvegarde la sélection courante sous un nom
  (écraser un groupe existant remplace ses hosts mais conserve ses tags) ;
- `ctrl-e` ouvre le picker : `Enter` connecte, la multi-sélection connecte
  à l'union des groupes, `ctrl-d` supprime le groupe sous le curseur ;
- `tmssh connect @prod-web` fonctionne aussi en CLI (mélange possible
  avec des hosts) ;
- le champ `tags` s'édite directement dans `hosts.json` (comme les icônes) ;
- un host disparu de `~/.ssh/config` est ignoré à la résolution ;
  l'historique enregistre les hosts individuels, jamais le groupe.

## Fichiers

- `~/.config/tmssh/hosts.json` — source de vérité des tags
- `~/.config/tmssh/history.json` — historique des connexions
- `~/.config/tmssh/backups/` — backups datés de `~/.ssh/config` (max 10, rotation auto)
- `~/.ssh/config` — source de vérité des paramètres de connexion ;
  les tags y sont persistés en commentaire `# tmssh:tags=prod,web`

`tmssh sync` réconcilie manuellement les deux fichiers (fait aussi
automatiquement à chaque modification de tag).

## Sous-commandes

```
tmssh              UI fzf
tmssh list         liste formatée (pour fzf)
tmssh connect H…   ouvre les connexions tmux (H = host ou @groupe)
tmssh tag add H…   prompt d'ajout de tag
tmssh tag rm H…    prompt de suppression de tag
tmssh tags         liste les tags connus
tmssh groups       liste les groupes de connexion
tmssh group save H…  sauvegarde un groupe (prompt du nom)
tmssh group rm G   supprime un groupe
tmssh group pick   picker de groupes (Enter = connexion)
tmssh tagfilter pick   ouvre le tag picker (filtre par tags, sémantique ET)
tmssh tagfilter clear  efface le filtre par tags
tmssh sync         sync ssh_config <-> hosts.json
tmssh info H       preview enrichi d'un host
tmssh history      historique
tmssh backups      liste les backups de ~/.ssh/config
```

## Roadmap / TODO

- [x] Filtre par tag dédié (tag picker, `ctrl-f`)
- [x] Icônes configurables dans hosts.json + icône par défaut
- [x] Backup automatique de `~/.ssh/config` avant toute sync
- [x] Preview enrichi par host (`ctrl-h` : latence TCP, tags, historique)
- [x] Groupes de connexion sauvegardés (ex. `prod-web`, `all-db`)
- [ ] ~~TUI native (bubbletea) à la place de fzf~~ (abandonné : l'architecture fzf + binaire ré-entrant est assumée)
- [ ] Include ssh_config (`Include ~/.ssh/config.d/*`)
- [ ] Tests unitaires du parseur ssh_config
