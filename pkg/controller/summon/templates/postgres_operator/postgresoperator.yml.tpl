apiVersion: db.ridecell.io/v1beta1
kind: PostgresOperatorDatabase
metadata:
 name: {{ .Instance.Name }}
 namespace: {{ .Instance.Namespace }}
spec:
  databaseRef:
    name: {{ .Instance.Spec.Database.SharedDatabaseName }}-database
  database: {{ .Instance.Name | replace "-" "_" }}
