```yaml
rules:
- runAsNonRoot:
    enabled: true
    replaceUser:
      enabled: true
      fromUser: 0
      toUser: 82
  conditions:
  - key: .ContainerType
    operator: equal
    value: container
  - key: .Namespace
    operator: regexp
    value: ^(prod|stage)$
```