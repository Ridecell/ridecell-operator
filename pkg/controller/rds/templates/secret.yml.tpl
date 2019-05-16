apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.rds-user-password
  namespace: {{ .Instance.Namespace }}
data: {}
