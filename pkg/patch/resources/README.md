```yaml
rules:
- addDefaultResources:
    enabled: true
  conditions:
  - key: .Namespace
    operator: regexp
    value: prod
```