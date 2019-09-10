{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "command" }}[python, -m, twisted, --log-format, text, web, --listen, tcp:8000, --wsgi, summon_platform.wsgi.application]{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.Replicas.Web | default 0 }}{{ end }}
{{ define "memory_limit" }}2G{{ end }}
{{ define "containerExtra" }}
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8000
          periodSeconds: 2
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 60
{{ end }}
{{ template "deployment" . }}
