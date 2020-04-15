{{ define "componentName" }}celeryd{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "target"}}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-celeryd{{ end }}
{{ define "metric" }}
        name: ridecell:rabbitmq_summon_celery_queue_scaler
        selector:
          matchLabels: 
            vhost: {{ .Instance.Name | quote }}
{{ define "mTarget" }}
        type: Value
        value: 1000{{ end }}
{{ template "hpa" . }}