apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ .Instance.Name }}-postgres-exporter
  namespace: {{ .Instance.Name }}
  labels:
    k8s-app: {{ .Instance.Name }}-postgres-exporter
    monitoredBy: prometheus-infra
spec:
  selector:
    matchLabels:
      app: postgres-exporter
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
