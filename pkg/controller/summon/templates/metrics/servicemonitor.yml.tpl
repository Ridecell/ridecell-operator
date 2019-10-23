apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ .Instance.Name }}-metrics
  namespace: {{ .Instance.Namespace }}
  labels:
    k8s-app: {{ .Instance.Name }}-metrics
    monitoredBy: prometheus-infra
spec:
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-metrics
  endpoints:
  - targetPort: 9000
    interval: 30s
    metricRelabelings:
    - action: replace
      sourceLabels:
      - instance
      targetLabel: instance
      replacement: {{ .Instance.Name }}
