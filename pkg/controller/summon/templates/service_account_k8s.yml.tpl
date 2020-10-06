apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::{{ .Extra.accountId }}:role/summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}"
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}