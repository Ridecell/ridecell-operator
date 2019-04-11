apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.rabbitmq-user-password
  namespace: {{ .Instance.Namespace }}
data: {}
