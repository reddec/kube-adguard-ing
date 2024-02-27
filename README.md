# kube-adguard-ing

Reflect Kuberenets [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
to [ADGuard](https://adguard.com/en/adguard-home/overview.html).

It watches for changes in `Ingress` and creates `A` records in ADGuard DNS with all domain used in
routes, and addresses from used LoadBalancer.

It also supports (optional) static list of domain and addresses if needed.

## Installation

```shell
curl -L https://github.com/reddec/kube-adguard-ing/releases/latest/download/kube-adguard-ing.yaml > kube-adguard-ing.yaml
# change secrets and URL ...
kubectl apply -f kube-adguard-ing.yaml
```

Manifests are prepared and uploaded for each release.

Examples see in [deployment](deployment).

## Dynamic Ingress

Almost any Ingress will be watched and exported.

For example:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example
spec:
  ingressClassName: nginx
  rules:
    - host: foo.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: foo-service
                port:
                  name: http
    - host: bar.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: foo-service
                port:
                  name: http
```

With Load Balancers IPs `1.2.3.4`, `5.6.7.8` will generate records

    foo.example.com -> 1.2.3.4
    foo.example.com -> 5.6.7.8
    bar.example.com -> 1.2.3.4
    bar.example.com -> 5.6.7.8

## Static

Since kube-adguard-ing manages records in adguard (which includes removing from ADGuard),
it could be required to add list of static domain records.
It can be done by mounting YAML file and setting `STATIC_PATH`.

For example:

```yaml
- domain: foo.example.com
  address:
    - 1.2.3.4
    - 5.6.7.8

- domain: "*.app.example.com"
  address:
    - 11.22.33.44
```

See example for kustomize in [deployment](deployment/kustomize).

## Environment

The required variables in cluster mode are:

- `ADGUARD_URL`
- `ADGUARD_USER`
- `ADGUARD_PASSWORD`

| Description                                    | Environment variable |
|------------------------------------------------|----------------------|
| Path to kubeconfig for local setup             | `KUBE_CONFIG`        |
| Kuberenetes master URL                         | `KUBE_URL`           |
| Minimal interval between updates (default: 3s) | `THROTTLE`           |
| Sync interval with kube (default: 1m)          | `SYNC_INTERVAL`      |
| Initial sync timeout (default: 10s)            | `TIMEOUT`            |
| AdGuard URL                                    | `ADGUARD_URL`        |
| Username                                       | `ADGUARD_USER`       |
| Password                                       | `ADGUARD_PASSWORD`   |
| Single operation timeout (default: 5s)         | `ADGUARD_TIMEOUT`    |
| Path to yaml file                              | `STATIC_PATH`        |
| YAML file cache duration (default: 5s)         | `STATIC_TTL`         |

## Usage

```
Application Options:
  -c, --kube-config=      Path to kubeconfig for local setup [$KUBE_CONFIG]
  -u, --kube-url=         Kuberenetes master URL [$KUBE_URL]
      --throttle=         Minimal interval between updates (default: 3s) [$THROTTLE]
      --sync-interval=    Sync interval with kube (default: 1m) [$SYNC_INTERVAL]
      --timeout=          Initial sync timeout (default: 10s) [$TIMEOUT]

AdGuard configuration:
      --adguard.url=      AdGuard URL [$ADGUARD_URL]
      --adguard.user=     Username [$ADGUARD_USER]
      --adguard.password= Password [$ADGUARD_PASSWORD]
      --adguard.timeout=  Single operation timeout (default: 5s) [$ADGUARD_TIMEOUT]

Static records:
      --static.path=      Path to yaml file [$STATIC_PATH]
      --static.ttl=       YAML file cache duration (default: 5s) [$STATIC_TTL]

Help Options:
  -h, --help              Show this help message
```

