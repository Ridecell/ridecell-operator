apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.dispatch
  namespace: {{ .Instance.Namespace }}
data: {}
