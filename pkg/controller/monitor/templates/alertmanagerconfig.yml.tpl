apiVersion: monitoring.ridecell.io/v1beta1
kind: AlertManagerConfig
metadata:
  name: alertmanagerconfig-{{  .Instance.Name  }}
  namespace: {{ .Instance.Namespace | quote }}
spec:
  alertManagerName: alertmanager-infra
  alertMangerNamespace: alertmanager
  routes: | 
    match_re:
      servicename: ".*{{ .Instance.Spec.ServiceName }}.*"
    routes:
{{ if .Extra.pd -}}
    - receiver: {{ .Extra.pd.Name }}
      match:
        severity: critical
      continue: true
{{ end -}}}
    - receiver: {{ .Extra.slack.Name }}
  receivers:
    - {{ .Extra.slack  | toJson  | quote}}
    {{ if .Extra.pd -}}
    - {{ .Extra.pd  | toJson  | quote }}
    {{ end -}}