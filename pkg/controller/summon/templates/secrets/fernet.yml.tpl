apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.fernet-keys
  namespace: {{ .Instance.Namespace }}
type: Opaque
data: {}
