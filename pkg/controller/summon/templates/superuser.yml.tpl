apiVersion: summon.ridecell.io/v1beta1
kind: DjangoUser
metadata:
  name: {{ .Instance.Name }}-dispatcher
  namespace: {{ .Instance.Namespace }}
spec:
  email: dispatcher@ridecell.com
  superuser: true
  database:
    {{- if .Instance.Spec.Database.ExclusiveDatabase }}
    host: {{ .Instance.Name }}-database.{{ .Instance.Namespace }}
    database: summon
    username: summon
    passwordSecretRef:
      name: summon.{{ .Instance.Name }}-database.credentials
    {{- else }}
    host: {{ .Instance.Spec.Database.SharedDatabaseName }}-database.{{ .Instance.Namespace }}
    database: {{ .Instance.Name | replace "-" "_" }}
    username: {{ .Instance.Name | replace "-" "_" }}
    passwordSecretRef:
      name: {{ .Instance.Name }}.{{ .Instance.Spec.Database.SharedDatabaseName }}-database.credentials
    {{- end }}
