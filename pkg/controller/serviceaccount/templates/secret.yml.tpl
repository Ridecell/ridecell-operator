apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.gcp-credentials
  namespace: {{ .Instance.Namespace }}
data:
  svc.json: {{ .Extra.serviceAccount }}
