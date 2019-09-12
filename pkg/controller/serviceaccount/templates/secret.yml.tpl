apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.gcp-credentials
  namespace: {{ .Instance.Namespace }}
data:
  google_service_account.json: {{ .Extra.serviceAccount }}
