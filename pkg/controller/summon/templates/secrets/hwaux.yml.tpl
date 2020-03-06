apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.hwaux
  namespace: {{ .Instance.Namespace }}
data: {}
