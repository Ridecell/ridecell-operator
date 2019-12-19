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
      initContainers:
      - name: copy-certs
        image: us.gcr.io/ridecell-1/postgres-exporter-certs:2
        command:
        - sh
        - -c
        - cp /*.pem /etc/ssl/certs
        volumeMounts:
        - name: certs
          mountPath: /etc/ssl/certs
      containers:
      - name: postgres-exporter
        image: wrouesnel/postgres_exporter:v0.4.7
        env:
        - name: DATA_SOURCE_URI
          value: "{{ .Extra.Conn.Host }}:{{ .Extra.Conn.Port | default 5432 }}/postgres?sslmode={{ .Extra.Conn.SSLMode | default "verify-full" }}"
        - name: DATA_SOURCE_USER
          value: {{ .Extra.Conn.Username }}
        - name: DATA_SOURCE_PASS
          valueFrom:
            secretKeyRef:
              name: {{ .Extra.Conn.PasswordSecretRef.Name }}
              key: {{ .Extra.Conn.PasswordSecretRef.Key | default "password" }}
        ports:
        - name: metrics
          containerPort: 9187
        volumeMounts:
        - name: certs
          mountPath: /etc/ssl/certs
      volumes:
      - name: certs
        emptyDir: {}
