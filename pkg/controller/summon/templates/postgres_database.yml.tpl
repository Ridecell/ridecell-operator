apiVersion: db.ridecell.io/v1beta1
kind: PostgresDatabase
metadata:
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: database
    app.kubernetes.io/instance: {{ .Instance.Name }}
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: database
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  extensions:
    postgis: ""
    postgis_topology: ""
