apiVersion: db.ridecell.io/v1beta1
kind: PostgresUser
metadata:
  name: {{ .Instance.Name }}-reporting
  namespace: {{ .Instance.Namespace }}
spec:
  username: reporting
  connection: {{ .Extra.Conn | toJson }}