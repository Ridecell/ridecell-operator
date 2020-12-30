{{ define "componentName" }}kafkaconsumer{{ end }}
{{ define "componentType" }}worker{{ end }}
{{- define "command" -}}
[python, /src/manage.py, run_kafka_consumer]
{{- end -}}
{{- define "deploymentPorts" -}}[]{{- end -}}
{{ define "metricsEnabled" }}"false"{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.Replicas.KafkaConsumer }}{{ end }}
{{ define "resources" }}{requests: {memory: "500M", cpu: "50m"}, limits: {memory: "500M"}}{{ end }}
{{ template "deployment" . }}
