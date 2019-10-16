{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "command" }}
{{- if .Instance.Spec.Metrics.Web -}}
[python, -m, summon_platform]
{{- else -}}
[python, -m, twisted, --log-format, text, web, --listen, tcp:8000, --wsgi, summon_platform.wsgi.application]
{{- end -}}
{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.Replicas.Web | default 0 }}{{ end }}
{{ define "memory_limit" }}2G{{ end }}
{{ define "containerExtra" }}
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8000
            httpHeaders:
            - name: X-Forwarded-Proto
              value: https
          periodSeconds: 2
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8000
            httpHeaders:
            - name: X-Forwarded-Proto
              value: https
          initialDelaySeconds: 60
{{ end }}
{{ template "deployment" . }}
