{{ define "componentName" }}kafkaconsumer{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "command" }}
[python, /src/manage.py, run_kafka_consumer]
{{ end }}
{{ define "deploymentPorts" }}
[{containerPort: 9000}]
{{ end }}
{{ define "metricsEnabled" }}"false"{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.Replicas.KafkaConsumer }}{{ end }}
{{ define "resources" }}{requests: {memory: "500M", cpu: "50m"}, limits: {memory: "500M"}}{{ end }}
{{ define "containerExtra" }}
        readinessProbe:
          httpGet:
            path: /healthz
            port: 9000
            httpHeaders:
            - name: X-Forwarded-Proto
              value: https
          periodSeconds: 2
        livenessProbe:
          httpGet:
            path: /healthz
            port: 9000
            httpHeaders:
            - name: X-Forwarded-Proto
              value: https
          initialDelaySeconds: 60
{{ end }}
{{ template "deployment" . }}
