apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.tripshare
  namespace: {{ .Instance.Namespace }}
data: {}
