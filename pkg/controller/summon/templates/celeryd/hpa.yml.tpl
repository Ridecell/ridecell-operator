{{ define "componentName" }}celeryd{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "target"}}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-celeryd{{ end }}
{{ define "metric" }}
        name: ridecell:rabbitmq_summon_queue_messages_ready
        selector:
          matchLabels: 
            vhost: {{ .Instance.Name | quote }}
            queue: "celery"{{ end }}
{{ define "mTarget" }}
        type: Value
        value: 2000{{ end }}
{{ template "hpa" . }}