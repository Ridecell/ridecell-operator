{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "extraAnnotations" }}
    monitor.ridecell.io/healthz-path: healthz
    monitor.ridecell.io/should-be-probed: "true"
{{ end }}
{{ define "servicePorts" }}
{{- if .Instance.Spec.Metrics.Web -}}
[{protocol: TCP, port: 8000}, {protocol: TCP, port: 9000, name: metrics}]
{{- else -}}
[{protocol: TCP, port: 8000}]
{{- end -}}
{{ end }}
{{ template "service" . }}
