apiVersion: db.ridecell.io/v1beta1
kind: RDSInstance
metadata:
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}
spec: {{ .Extra.DbConfig.Spec.Postgres.RDS | toJson }}
