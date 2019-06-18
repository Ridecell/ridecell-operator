apiVersion: monitoring.ridecell.io/v1beta1
kind: AlertManagerConfig
metadata:
  name: alertmanagerconfig-{{  .Instance.Name  }}
  namespace: {{ .Instance.Namespace | quote }}
spec:
  alertManagerName: alertmanager-infra
  alertMangerNamespace: alertmanager
  data: 
    routes: {{ .Extra.routes  }}
    receiver:  {{ .Extra.receiver  }}
