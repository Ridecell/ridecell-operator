apiVersion: db.ridecell.io/v1beta1
kind: DjangoUser
metadata:
  name: {{ .Instance.Name }}-dispatcher
  namespace: {{ .Instance.Namespace }}
spec:
  email: dispatcher@ridecell.com
  superuser: true
  database:
    host: {{ .Instance.Name }}-database.{{ .Instance.Namespace }}
    username: summon
    database: summon
    passwordSecretRef:
      name: summon.{{ .Instance.Name }}-database.credentials
