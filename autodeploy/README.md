# autodeploy

Keeps one Kubernetes deployment on the newest release of its image. It runs in the cluster, and every few minutes it lists the image's tags on the registry. When a `vX.Y.Z` tag is newer than the one the deployment runs, it sets that tag on the deployment, and Kubernetes rolls it out.

- **Pull-based:** nothing outside the cluster needs access to it.
- **Least privilege:** the service account needs only `get` and `patch` on the one deployment.
- **Only plain releases:** `v1.2.3` counts. `latest`, `sha-…` and pre-releases like `v1.3.0-rc.1` are ignored.
- **Never downgrades.** To roll back, release the older commit again under a new, higher version, or pause it (below) and set the image by hand.
- **Leaves other images alone:** a container running anything but a tag of `AUTODEPLOY_IMAGE` is reported and not changed.

Registries that use the token flow of the OCI distribution API work, such as GHCR. Credentials come from a Docker `config.json`, such as a mounted image pull secret. It is read on every check, so a renewed secret takes effect without a restart. Public images need no credentials.

## Configuration

| Variable | Default | |
|---|---|---|
| `AUTODEPLOY_IMAGE` | (required) | Image without a tag, e.g. `ghcr.io/owner/app` |
| `AUTODEPLOY_DEPLOYMENT` | (required) | Deployment to update |
| `AUTODEPLOY_CONTAINER` | the deployment's name | Container in the deployment |
| `AUTODEPLOY_NAMESPACE` | the pod's own namespace | |
| `AUTODEPLOY_INTERVAL` | `5m` | Time between checks, at least `10s` |
| `AUTODEPLOY_AUTH_FILE` | none | Docker `config.json` with registry credentials |

Logs are JSON on stdout. A check that changes nothing logs at debug level, except when the state changes, so a quiet deployment stays quiet.

## Pausing

```sh
kubectl -n app annotate deployment/app autodeploy/paused=true --overwrite
kubectl -n app annotate deployment/app autodeploy/paused-
```

## Example

```yaml
apiVersion: v1
kind: ServiceAccount
metadata: { name: autodeploy, namespace: app }
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata: { name: autodeploy, namespace: app }
rules:
  - apiGroups: [apps]
    resources: [deployments]
    resourceNames: [app]
    verbs: [get, patch]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata: { name: autodeploy, namespace: app }
subjects: [{ kind: ServiceAccount, name: autodeploy, namespace: app }]
roleRef: { apiGroup: rbac.authorization.k8s.io, kind: Role, name: autodeploy }
---
apiVersion: apps/v1
kind: Deployment
metadata: { name: autodeploy, namespace: app }
spec:
  replicas: 1
  selector: { matchLabels: { app: autodeploy } }
  template:
    metadata: { labels: { app: autodeploy } }
    spec:
      serviceAccountName: autodeploy
      containers:
        - name: autodeploy
          image: ghcr.io/sander-skjulsvik/tools/autodeploy:v0.4.0
          env:
            - { name: AUTODEPLOY_IMAGE, value: ghcr.io/owner/app }
            - { name: AUTODEPLOY_DEPLOYMENT, value: app }
            - { name: AUTODEPLOY_AUTH_FILE, value: /registry/config.json }
          volumeMounts: [{ name: registry, mountPath: /registry, readOnly: true }]
      volumes:
        - name: registry
          secret:
            secretName: registry-pull # type kubernetes.io/dockerconfigjson
            items: [{ key: .dockerconfigjson, path: config.json }]
```

Network policies: it needs to reach the Kubernetes API server and the registry, and nothing needs to reach it.
