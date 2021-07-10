{{ define "componentName" }}celeryd{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "target"}}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-celeryd
{{- end }}
{{ define "minReplicas" }}{{ .Instance.Spec.Replicas.CelerydAuto.Min }}{{ end }}
{{ define "maxReplicas" }}{{ .Instance.Spec.Replicas.CelerydAuto.Max }}{{ end }}
{{ define "metric" }}
        name: ridecell:rabbitmq_summon_celery_queue_scaler
        selector:
          matchLabels: 
            vhost: {{ .Instance.Name | quote }}
{{- end }}
{{ define "mTarget" }}
        type: Value
        value: 1
{{- end }}
{{ template "hpa" . }}