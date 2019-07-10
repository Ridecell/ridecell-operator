apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ .Instance.Name }}-postgres-exporter
  namespace: {{ .Instance.Namespace }}
  labels:
    k8s-app: postgres-exporter
    monitoredBy: prometheus-infra
spec:
  selector:
    matchLabels:
      app: postgres-exporter
      instance: {{ .Instance.Name }}
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
