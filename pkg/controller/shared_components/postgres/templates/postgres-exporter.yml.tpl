apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-postgres-exporter
  namespace: {{ .Instance.Namespace }}
  labels:
    app: postgres-exporter
    instance: {{ .Instance.Name }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres-exporter
      instance: {{ .Instance.Name }}
  template:
    metadata:
      labels:
        app: postgres-exporter
        instance: {{ .Instance.Name }}
    spec:
      containers:
      - name: postgres-exporter
        image: wrouesnel/postgres_exporter:v0.4.7
        env:
        - name: DATA_SOURCE_URI
          value: "{{ .Extra.Conn.Host }}:{{ .Extra.Conn.Port }}/?sslmode={{ .Extra.Conn.SSLMode | default "verify-full" }}"
        - name: DATA_SOURCE_USER
          value: {{ .Extra.Conn.Username }}
        - name: DATA_SOURCE_PASS
          valueFrom:
            secretKeyRef:
              name: {{ .Extra.Conn.PasswordSecretRef.Name }}
              key: {{ .Extra.Conn.PasswordSecretRef.Key }}
        ports:
        - name: metrics
          containerPort: 9187
