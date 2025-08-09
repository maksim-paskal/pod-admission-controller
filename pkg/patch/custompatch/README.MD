```yaml
rules:
- custompatches:
  - op: "remove"
    path: "/spec/affinity"
  - op: "remove"
    path: "/spec/nodeSelector"
  - op: "remove"
    path: "{{ .PodContainer.ContainerPath }}/resources"
  - op: "add"
    # Use ` for strings in templates
    value: "{{ default `a` .OwnerName }}"
```
