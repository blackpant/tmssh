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
| `ctrl-h` | afficher l'historique des connexions |

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

## Fichiers

- `~/.config/tmssh/hosts.json` — source de vérité des tags
- `~/.config/tmssh/history.json` — historique des connexions
- `~/.ssh/config` — source de vérité des paramètres de connexion ;
  les tags y sont persistés en commentaire `# tmssh:tags=prod,web`

`tmssh sync` réconcilie manuellement les deux fichiers (fait aussi
automatiquement à chaque modification de tag).

## Sous-commandes

```
tmssh              UI fzf
tmssh list         liste formatée (pour fzf)
tmssh connect H…   ouvre les connexions tmux
tmssh tag add H…   prompt d'ajout de tag
tmssh tag rm H…    prompt de suppression de tag
tmssh tags         liste les tags connus
tmssh sync         sync ssh_config <-> hosts.json
tmssh history      historique
```

## Roadmap / TODO

- [x] Filtre par tag dédié (tag picker, `ctrl-f`)
- [x] Icônes configurables dans hosts.json + icône par défaut
- [ ] Include ssh_config (`Include ~/.ssh/config.d/*`)
- [ ] Tests unitaires du parseur ssh_config
