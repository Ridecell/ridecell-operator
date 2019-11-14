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
    pg_trgm: ""
  dbConfigRef: {{ .Instance.Spec.Database.DbConfigRef | toJson }}
  {{ if .Instance.Spec.MigrationOverrides.PostgresDatabase }}
  databaseName: {{ .Instance.Spec.MigrationOverrides.PostgresDatabase }}
  {{ end }}
  {{ if .Instance.Spec.MigrationOverrides.PostgresUsername }}
  owner: {{ .Instance.Spec.MigrationOverrides.PostgresUsername }}
  {{ end }}
  {{ if .Instance.Spec.MigrationOverrides.RDSInstanceID }}
  migrationOverrides:
    rdsInstanceId: {{ .Instance.Spec.MigrationOverrides.RDSInstanceID }}
    {{ if .Instance.Spec.MigrationOverrides.RDSMasterUsername }}
    rdsMasterUsername: {{ .Instance.Spec.MigrationOverrides.RDSMasterUsername }}
    {{ end }}
  {{ end }}

