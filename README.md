# Torrent Operator 🏴‍☠️

A Kubernetes Operator for managing Torrent searches and requests similarly to how `cert-manager` handles certificates. It bridges the gap between your GitOps workflow and Torrent Indexers.

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Kubernetes](https://img.shields.io/badge/kubernetes-%23326ce5.svg?style=for-the-badge&logo=kubernetes&logoColor=white)

## 🚀 Features

- **GitOps Friendly**: Everything is a CRD (`TorrentRequest`, `Indexer`, `Torrent`).
- **Indexer Support**: Compatible with generic HTML parsers and Prowlarr-style definitions.
- **FlareSolverr Integration**: Built-in support for bypassing Cloudflare protection on indexers.
- **ArgoCD Ready**: Implements standard Conditions and OwnerReferences for visual feedback in ArgoCD.
- **Observability**: Exports Prometheus metrics (`torrent_searches_total`, `torrent_request_duration_seconds`).

## 🛠 Architecture

1.  **Define Indexers**: Configure your torrent sites as `Indexer` resources.
2.  **Request a Torrent**: Create a `TorrentRequest` with keywords.
3.  **Controller Action**:
    - Queries all healthy indexers (optionally via FlareSolverr).
    - Parses HTML results using CSS selectors.
    - Filters by `minSeeders`.
    - Selects the best match.
4.  **Result**: A `Torrent` resource is created with the magnet link, fully linked to the original request.

## 📦 Installation

**1. Deploy the Operator**

```bash
make deploy
```

**2. (Optional) Configure FlareSolverr**

If you need to access sites protected by Cloudflare, you can deploy [FlareSolverr](https://github.com/FlareSolverr/FlareSolverr) and update the operator deployment to start with the `--flaresolverr-url` flag pointing to the service (e.g., `http://flaresolverr:8191`).

## 📝 Usage

### 1. Configure an Indexer

```yaml
apiVersion: torrents.vitoru.fun/v1alpha1
kind: Indexer
metadata:
  name: torrentdownload
spec:
  links:
    - "https://www.torrentdownload.info"
  search:
    paths:
      - path: "/search?q={{ .Keywords }}"
    rows:
      selector: "table.table-striped tbody tr"
    fields:
      title:
        selector: "td:nth-child(2) a"
      download:
        selector: "td:nth-child(4) a"
        attribute: "href"
      size:
        selector: "td:nth-child(5)"
      seeders:
        selector: "td:nth-child(6)"
      leechers:
        selector: "td:nth-child(7)"
```

### 2. Request a Torrent

```yaml
apiVersion: torrents.vitoru.fun/v1alpha1
kind: TorrentRequest
metadata:
  name: ubuntu-iso
spec:
  keywords: "ubuntu 22.04"
  minSeeders: 10
```

### 3. Check Status

```bash
kubectl get torrentrequest ubuntu-iso
# NAME         STATE       READY   FOUND TORRENT
# ubuntu-iso   Completed   True    ubuntu-22-04-desktop-amd64
```

## 📊 Metrics

The operator exposes Prometheus metrics at `:8080/metrics`:

- `torrent_searches_total{indexer, status}`: Search volume and success/failure rates.
- `torrent_request_duration_seconds{indexer}`: Histogram of time taken to successfully find a torrent.
- `torrent_request_failure_duration_seconds`: Histogram of time taken for a TorrentRequest to fail (when no torrents are found).
- `torrents_created_total{indexer}`: Counter of successfully created torrents.
- `torrent_requests_failed_total`: Counter of failed torrent requests.

## 🤝 Contributing

**Manager**:

```bash
make run
```

**Tests**:

```bash
make test
```

---

_Built with ❤️ using Kubebuilder_
