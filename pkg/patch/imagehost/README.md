```yaml
rules:
- replaceContainerImageHost:
    enabled: true
    to: '{{ env "CLUSTER_REGISTRY" }}'
  conditions:
  - key: .Image.Domain
    operator: regexp
    value: ^(localhost:32000|10.100.0.11:5000|registry.(.+).com)$

- replaceContainerImageHost:
    enabled: true
    to: '{{ env "CLUSTER_REGISTRY_HUB" }}'
  conditions:
  - key: .Image.Domain
    operator: regexp
    value: ^(docker-hub-proxy.+|docker.io)$
```