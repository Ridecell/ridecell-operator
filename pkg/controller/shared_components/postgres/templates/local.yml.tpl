apiVersion: acid.zalan.do/v1
kind: postgresql
metadata:
  name: {{ .Instance.Name }}-database
  namespace: {{ .Instance.Namespace }}
spec:
  {{/* Copy over fields. */}}
  {{ $local := .Extra.DbConfig.Spec.Postgres.Local }}
  postgresql:
    version: {{ $local.PostgresqlParam.PgVersion | quote }}
    parameters: {{ $local.PostgresqlParam.Parameters | toJson }}
  volume: {{ $local.Volume | toJson }}
  resources: {{ $local.Resources | toJson }}
  dockerImage: {{ $local.DockerImage | quote }}
  enableMasterLoadBalancer: {{ $local.EnableMasterLoadBalancer | toJson }}
  enableReplicaLoadBalancer: {{ $local.EnableReplicaLoadBalancer | toJson }}
  allowedSourceRanges: {{ $local.AllowedSourceRanges | toJson }}
  maintenanceWindows: {{ $local.MaintenanceWindows | toJson }}
  clone: {{ $local.Clone | toJson }}
  databases: {{ $local.Databases | toJson }} {{/* Handled by us, so usually empty */}}
  tolerations: {{ $local.Tolerations | toJson }}
  sidecars: {{ $local.Sidecars | toJson }}
  pod_priority_class_name: {{ $local.PodPriorityClassName | quote }}

  {{/* Fields we care about */}}
  teamId: {{ .Instance.Name }}
  numberOfInstances: {{ $local.NumberOfInstances | default 2 }}
  users:
    {{ range $user, $flags := $local.Users }}
    {{ $user }}: {{ $flags | toJson }}
    {{ end }}
    ridecell-admin: [superuser]
