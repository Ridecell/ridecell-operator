{{ define "componentName" }}metrics{{ end }}
{{ define "componentType" }}metrics{{ end }}
{{ define "extraLabels" }}metrics-enabled: "true"{{ end }}
{{ define "servicePorts" }}[{protocol: TCP, port: 9000}]{{ end }}
{{ define "selectors" }}{app.kubernetes.io/part-of: {{ .Instance.Name }}, metrics-enabled: "true"}{{ end }}
{{ template "service" . }}
