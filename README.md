# gluetun-hc

Moniteur de connexion pour Gluetun. Il tourne dans Docker, surveille Gluetun et une liste de conteneurs en `network_mode: service:gluetun`, et redémarre ceux qui ont perdu leur réseau.

## Pourquoi

Quand Gluetun redémarre ou que son tunnel WireGuard tombe, les conteneurs qui partagent son namespace réseau perdent leur interface. Ils restent `Up` sans connexion jusqu'à un redémarrage manuel.

## Fonctionnement

À chaque intervalle :

1. **Gluetun** : `ping` exécuté dans le conteneur Gluetun. Après `FAIL_THRESHOLD` échecs consécutifs, Gluetun est redémarré. Quand il répond `RISE_THRESHOLD` fois de suite (après son délai de grâce), tous les conteneurs surveillés sont redémarrés une fois ; un redémarrage refusé est retenté au contrôle suivant. Tant que Gluetun est absent ou arrêté, rien n'est contrôlé ni redémarré.
2. **Conteneurs** : `ping` exécuté dans chaque conteneur surveillé (sortant), seulement si Gluetun répond.
3. **Entrant** (facultatif) : connexion TCP vers `IP publique:port forwardé` de Gluetun. En cas d'échec, `INBOUND_CONTAINER` est redémarré. Sans port forwardé ou API de contrôle injoignable, le contrôle est sauté.

Un conteneur arrêté à la main n'est jamais relancé : l'état est vérifié avant tout redémarrage, y compris celui de Gluetun.

Après un redémarrage, la cible n'est plus testée pendant son délai de grâce.

## Configuration

| Variable | Défaut | Rôle |
|---|---|---|
| `GLUETUN_CONTAINER` | obligatoire | Nom du conteneur Gluetun |
| `WATCHED` | obligatoire | Conteneurs surveillés, `nom[:grâce]` séparés par des virgules (ex. `app:10m,autre`) |
| `INBOUND_CONTAINER` | vide | Conteneur qui écoute sur le port forwardé (doit figurer dans `WATCHED`) |
| `PING_TARGETS` | `9.9.9.9,1.1.1.1` | Cibles de ping, un succès suffit |
| `INTERVAL` | `10s` | Intervalle entre deux contrôles |
| `FAIL_THRESHOLD` / `RISE_THRESHOLD` | `3` / `3` | Échecs consécutifs avant redémarrage ; pings réussis consécutifs de Gluetun avant de raccrocher les conteneurs |
| `GRACE` | `5m` | Délai de grâce par défaut après un redémarrage |
| `GLUETUN_GRACE` | valeur de `GRACE` | Délai de grâce de Gluetun |
| `GLUETUN_CONTROL_URL` | `http://gluetun:8000` | API de contrôle de Gluetun |
| `GLUETUN_API_KEY` | vide | Clé d'API (en-tête `X-API-Key`) |

Le rôle d'authentification de Gluetun doit autoriser `GET /v1/publicip/ip` et `GET /v1/portforward`.

## Droits Docker

Le moniteur utilise le socket Docker pour lister, exécuter (`exec`) et redémarrer des conteneurs. Un accès direct au socket équivaut à un accès root à l'hôte : préférer un proxy de socket qui n'autorise que ces opérations, via `DOCKER_HOST`.

## Limites

- Le binaire `ping` doit exister dans l'image de chaque conteneur surveillé ; sinon la sonde est ignorée et journalisée.
- Un simple redémarrage suffit pour raccrocher un conteneur. Si Gluetun est recréé (nouvel identifiant), les conteneurs dépendants doivent être recréés par Compose.

## Développement

```bash
go vet ./...
go test -p 4 ./...
docker build -t gluetun-hc .
```
