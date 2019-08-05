apiVersion: monitoring.ridecell.io/v1beta1
kind: AlertManagerConfig
metadata:
  name: alertmanagerconfig-{{  .Instance.Name  }}
  namespace: {{ .Instance.Namespace | quote }}
spec:
  alertManagerName: alertmanager-infra
  alertMangerNamespace: alertmanager
  data: 
    routes: |
      receiver: {{ .Instance.Name }}
      group_by:
      - alertname
      match_re:
        servicename: ".*{{ .Instance.Spec.ServiceName }}.*"
    receiver:  {{ .Extra.receiver  | toJson }}