apiVersion: db.ridecell.io/v1beta1
kind: PostgresExtension
metadata:
  name: {{ .Instance.Name }}-{{ .Extra.ExtensionName | replace "_" "-" }}
  namespace: {{ .Instance.Namespace | quote }}
spec:
  extensionName: {{ .Extra.ExtensionName | quote }}
  version: {{ .Extra.ExtensionVersion | quote }}
  database: {{ .Extra.ExtensionConn | toJson }}
