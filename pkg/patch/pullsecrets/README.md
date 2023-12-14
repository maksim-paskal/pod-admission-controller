```yaml
rules:
- imagePullSecrets:
  - name: docker-registry-secret
  conditions:
  - key: env "SENTRY_ENVIRONMENT"
    operator: equal
    value: azure-dev
```