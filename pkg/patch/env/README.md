```yaml
rules:
- env:
  - name: SENTRY_DSN
    value: '{{ GetSentryDSN .Image.Slug }}'
  - name: SENTRY_ENVIRONMENT
    value: '{{ .NamespaceLabels.environment }}'
  - name: SENTRY_RELEASE
    value: '{{ .Image.Tag }}'
  conditions:
  - key: 'GetSentryDSN .Image.Slug'
    operator: regexp
    value: .+
  - key: .Namespace
    operator: regexp
    value: ^(paket|romantic|cfaas)($|-main-.+)
```