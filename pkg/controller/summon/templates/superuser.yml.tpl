apiVersion: summon.ridecell.io/v1beta1
kind: DjangoUser
metadata:
  name: {{ .Instance.Name }}-dispatcher
  namespace: {{ .Instance.Namespace }}
spec:
  email: dispatcher@ridecell.com
  superuser: true
  database: {{ .Instance.Status.PostgresConnection | toJson }}
