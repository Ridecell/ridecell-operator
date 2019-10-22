apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.tenant-otakeys
  namespace: {{ .Instance.Namespace }}
data: {}
