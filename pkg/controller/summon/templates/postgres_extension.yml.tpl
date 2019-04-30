apiVersion: db.ridecell.io/v1beta1
kind: PostgresExtension
metadata:
  name: {{ .Instance.Name }}-{{ .Extra.ObjectName }}
  namespace: {{ .Instance.Namespace }}
spec:
  extensionName: {{ .Extra.ExtensionName }}
  database: {{ .Instance.Status.PostgresConnection | toJson }}
