```yaml
rules:
- tolerations:
  - key: "kubernetes.azure.com/scalesetpriority"
    operator: "Equal"
    value: "spot"
    effect: "NoSchedule"
  conditions:
  - key: env "SENTRY_ENVIRONMENT"
    operator: equal
    value: azure-dev
```