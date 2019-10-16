{{ define "serviceMonitor" }}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
  namespace: {{ .Instance.Namespace }}
  labels:
    k8s-app: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    monitoredBy: prometheus-infra
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
  endpoints:
  - port: metrics
    interval: 30s
{{ end }}
