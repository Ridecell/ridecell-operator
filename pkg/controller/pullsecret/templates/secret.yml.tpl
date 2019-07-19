apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Spec.PullSecretName }}
  namespace: {{ .Instance.Namespace }}
