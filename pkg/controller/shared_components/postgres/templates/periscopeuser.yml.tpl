apiVersion: db.ridecell.io/v1beta1
kind: PostgresUser
metadata:
  name: {{ .Instance.Name }}-periscope
  namespace: {{ .Instance.Namespace }}
spec:
  username: periscope
  connection: {{ .Extra.conn | toJson }}