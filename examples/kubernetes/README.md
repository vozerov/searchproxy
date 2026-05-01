# Kubernetes example

Deploys `searchproxy` as a small Deployment + Service in the current
namespace. The proxy listens on `5432` and `6379` and forwards to
`postgres:6379` / `redis:6379` — both resolved through the pod's
`search` list, so the same manifest works in any namespace where
those services exist.

```bash
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

Probes target the health server on `:8081`:

- `livenessProbe` → `GET /healthz`
- `readinessProbe` → `GET /readyz` (returns 200 only after every TCP
  listener is up)
