apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.postgres-user-password
  namespace: {{ .Instance.Namespace }}
data: {}
