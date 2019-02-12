apiVersion: db.ridecell.io/v1beta1
kind: PostgresExtension
metadata:
  name: {{ .Instance.Name }}-{{ .Extra.ObjectName }}
  namespace: {{ .Instance.Namespace }}
spec:
  extensionName: {{ .Extra.ExtensionName }}
  database:
    username: ridecell-admin
    {{- if .Instance.Spec.Database.ExclusiveDatabase }}
    host: {{ .Instance.Name }}-database.{{ .Instance.Namespace }}
    database: summon
    passwordSecretRef:
      name: ridecell-admin.{{ .Instance.Name }}-database.credentials
    {{- else }}
    host: {{ .Instance.Spec.Database.SharedDatabaseName }}-database.{{ .Instance.Namespace }}
    database: {{ .Instance.Name | replace "-" "_" }}
    passwordSecretRef:
      name: ridecell-admin.{{ .Instance.Spec.Database.SharedDatabaseName }}-database.credentials
    {{- end }}
