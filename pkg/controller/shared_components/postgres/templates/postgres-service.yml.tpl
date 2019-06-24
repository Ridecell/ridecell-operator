apiVersion: v1
kind: Service
metadata:
  name: {{ .Instance.Name }}-postgres-exporter
  namespace: {{ .Instance.Namespace }}
  labels:
    app: postgres-exporter
    instance: {{ .Instance.Name }}
spec:
  selector:
    app: postgres-exporter
    instance: {{ .Instance.Name }}
  ports:
    - port: 9187
      targetPort: 9187
      name: metrics
