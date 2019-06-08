{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "command" }}[python, -m, twisted, --log-format, text, web, --listen, tcp:8000, --wsgi, summon_platform.wsgi.application]{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.WebReplicas }}{{ end }}
{{ define "memory_limit" }}2G{{ end }}
{{ template "deployment" . }}
