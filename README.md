# gluetun-hc

Moniteur de connexion pour Gluetun. Il tourne dans Docker, surveille Gluetun et une liste de conteneurs en `network_mode: service:gluetun`, et redémarre ceux qui ont perdu leur réseau.

## Pourquoi

Quand Gluetun redémarre ou que son tunnel WireGuard tombe, les conteneurs qui partagent son namespace réseau perdent leur interface. Ils restent `Up` sans connexion jusqu'à un redémarrage manuel.

## Fonctionnement

À chaque intervalle :

1. **Gluetun** : `ping` exécuté dans le conteneur Gluetun. Après `FAIL_THRESHOLD` échecs consécutifs, Gluetun est redémarré. Quand il répond `RISE_THRESHOLD` fois de suite (après son délai de grâce), tous les conteneurs surveillés sont redémarrés une fois ; un redémarrage refusé est retenté au contrôle suivant. Tant que Gluetun est absent ou arrêté, rien n'est contrôlé ni redémarré.
2. **Conteneurs** : `ping` exécuté dans chaque conteneur surveillé (sortant), seulement si Gluetun répond.
3. **Entrant** (facultatif) : `nc -z` exécuté dans le conteneur Gluetun, donc dans le namespace du VPN, vers `IP publique:port forwardé`. Le moniteur n'ouvre aucune connexion lui-même et rien ne sort hors du tunnel. En cas d'échec, `INBOUND_CONTAINER` est redémarré. Sans port forwardé, avec une API ou des fichiers illisibles, ou sans `nc` dans l'image, le contrôle est sauté.

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
| `GLUETUN_PUBLIC_IP_FILE` | `/tmp/gluetun/ip` | Fichier de l'IP publique, lu dans le conteneur Gluetun |
| `GLUETUN_PORT_FILE` | `/tmp/gluetun/forwarded_port` | Fichier du port forwardé, lu dans le conteneur Gluetun |
| `GLUETUN_SOURCE` | `auto` | Origine de l'IP publique et du port : `auto` (fichiers puis API), `api` (API seule), `files` (fichiers seuls) |
| `GLUETUN_CONTROL_URL` | `http://gluetun:8000` | API de contrôle de Gluetun |
| `GLUETUN_API_KEY` | vide | Clé d'API (en-tête `X-API-Key`) |
| `GLUETUN_USER` / `GLUETUN_PASSWORD` | vides | Authentification basique, exclusive avec la clé d'API |

L'IP publique et le port forwardé sont lus d'abord dans les fichiers de Gluetun (`cat` exécuté dans son conteneur : rien à monter, aucune authentification). Si un fichier manque ou est illisible, le moniteur se rabat sur l'API de contrôle : son rôle d'authentification doit alors autoriser `GET /v1/publicip/ip` et `GET /v1/portforward`. Gluetun annonce la suppression de ces fichiers en v4.

## Mise à jour du port de la cible

Quand Gluetun obtient un port forwardé, le moniteur peut l'appliquer à une application (une seule cible, un seul port), par exemple le port d'écoute de qBittorrent.

- Il lit le port de Gluetun (fichier ou API, voir `GLUETUN_SOURCE`) à chaque contrôle.
- **qBittorrent** : il lit d'abord le port d'écoute actuel et n'écrit que s'il diffère. Un qBittorrent redémarré avec une configuration perdue est donc corrigé sans autre déclencheur.
- **Transmission** : même principe que qBittorrent, via son RPC (`peer-port`), avec la poignée de main d'identifiant de session gérée par la bibliothèque. `PORT_UPDATE_URL` peut être l'adresse de base (`http://transmission:9091`, le chemin `/transmission/rpc` est ajouté) ou l'URL complète du RPC. Désactiver `peer-port-random-on-start` dans la configuration de Transmission, sinon il retire le port au démarrage.
- **HTTP générique** : pas de lecture possible, le port est appliqué au démarrage du moniteur, à chaque changement et après chaque redémarrage de la cible ; un échec est retenté au contrôle suivant.
- Tant que le port n'est pas appliqué, le contrôle entrant est suspendu : la cible n'est jamais redémarrée à cause d'un port qui n'écoute pas encore.

| Variable | Défaut | Rôle |
|---|---|---|
| `PORT_UPDATE_MODE` | vide (désactivé) | `qbittorrent`, `transmission` ou `http` |
| `PORT_UPDATE_URL` | obligatoire | qBittorrent : URL de l'interface web (ex. `http://qbittorrent:8080`) ; `transmission` : adresse du RPC ; `http` : URL de la requête, `{{port}}` est remplacé |
| `PORT_UPDATE_CONTAINER` | `INBOUND_CONTAINER` | Conteneur cible (doit figurer dans `WATCHED`) |
| `PORT_UPDATE_USER` / `PORT_UPDATE_PASSWORD` | vides | qBittorrent et Transmission : identifiants de l'interface web ou du RPC |
| `PORT_UPDATE_METHOD` | `POST` | `http` : méthode |
| `PORT_UPDATE_HEADERS` | vide | `http` : en-têtes, un `Nom: valeur` par ligne, `{{port}}` remplacé |
| `PORT_UPDATE_BODY` | vide | `http` : corps, `{{port}}` remplacé |

L'interface web de la cible est jointe depuis le réseau Docker : elle partage le namespace de Gluetun, donc son nom d'hôte est celui du conteneur Gluetun (ex. `http://gluetun:8080`).

## Santé du conteneur

L'image déclare un `HEALTHCHECK` Docker : le conteneur `gluetun-hc` est `healthy` quand le dernier `ping` exécuté dans Gluetun vers une IP publique (`PING_TARGETS`, jamais un nom DNS) a réussi il y a moins de trois intervalles. Il est `unhealthy` avant la première sonde, pendant le délai de grâce qui suit un redémarrage de Gluetun, et dès qu'une sonde échoue, même sous le seuil de redémarrage (Docker applique ses propres tentatives : `--retries=3`).

Un autre service peut attendre cet état :

```yaml
depends_on:
  gluetun-hc:
    condition: service_healthy
```

Le moniteur écoute sur `HEALTH_ADDRESS` (défaut `127.0.0.1:8081`, joignable seulement depuis le conteneur) et la commande `gluetun-hc healthcheck` l'interroge.

## Intégration continue

`.github/workflows/ci.yml` lance `go vet` et `go test` à chaque push et pull request, puis construit l'image en `linux/amd64` et `linux/arm64`. Hors pull request, l'image est publiée sur `ghcr.io/<propriétaire>/gluetun-hc` : `latest` sur `main`, la version sur un tag `v*`, et le SHA du commit.

## Droits Docker

Le moniteur utilise l'API Docker pour inspecter, exécuter (`exec`) et redémarrer des conteneurs. Il lit `DOCKER_HOST` : l'exemple `compose.yaml` passe par un proxy de socket (`wollomatic/socket-proxy`) qui n'autorise que ces appels, et uniquement sur Gluetun et les conteneurs surveillés. Un accès direct au socket équivaut à un accès root à l'hôte.

- `DOCKER_GID` (identifiant du groupe `docker` de l'hôte) doit être défini : `DOCKER_GID=$(stat -c %g /var/run/docker.sock)`.
- Les expressions `-allowGET` et `-allowPOST` du proxy nomment les conteneurs autorisés (`gluetun`, `app`) : les adapter à `GLUETUN_CONTAINER` et `WATCHED`. Le proxy autorise aussi `GET /containers/json` (liste) pour la résolution des noms.
- `GLUETUN_CONTAINER` et `WATCHED` acceptent un nom de conteneur ou un nom de service Compose (label `com.docker.compose.service`). Un nom de service est résolu dans le projet Compose du moniteur (lu depuis son propre conteneur, via son nom d'hôte, donc ne pas définir `hostname:` sur lui). Si le projet est inconnu et que le nom correspond à plusieurs conteneurs, il est refusé : utiliser le nom du conteneur. Les règles du proxy portent sur le nom réel du conteneur, d'où les `container_name` de l'exemple.
- Le réseau `docker-proxy` est interne : le proxy n'est joignable que par le moniteur.

## Limites

- Le contrôle entrant part du tunnel vers l'IP publique du serveur VPN : il ne réussit que si le fournisseur renvoie ce trafic (retour par le même serveur). Si ce n'est pas le cas, il échoue en permanence et redémarre `INBOUND_CONTAINER` : ne pas définir `INBOUND_CONTAINER` dans ce cas.
- Le binaire `ping` doit exister dans l'image de chaque conteneur surveillé ; sinon la sonde est ignorée et journalisée.
- Un simple redémarrage suffit pour raccrocher un conteneur. Si Gluetun est recréé (nouvel identifiant), les conteneurs dépendants doivent être recréés par Compose.

## Développement

```bash
go vet ./...
go test -p 4 ./...
GLUETUN_HC_INTEGRATION=1 go test -p 4 ./internal/docker/...   # démarre un conteneur alpine jetable
docker build -t gluetun-hc .
```

## Licence

MIT, voir `LICENSE`.
