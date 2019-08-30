apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.saml
  namespace: {{ .Instance.Namespace }}
data: {}
