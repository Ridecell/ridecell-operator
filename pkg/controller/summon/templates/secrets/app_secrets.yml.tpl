apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.app-secrets
  namespace: {{ .Instance.Namespace }}
type: Opaque
data: {}
