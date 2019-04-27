apiVersion: db.ridecell.io/v1beta1
kind: PostgresUser
metadata:
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}
spec:
  username: {{ .Instance.Spec.Owner | quote }}
  connection: {{ .Instance.Status.AdminConnection | toJson }}
