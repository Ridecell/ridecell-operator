{{ define "componentName" }}static{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "metricsEnabled" }}"false"{{ end }}
{{ define "command" }}[caddy, "-port", "8000", "-root", /var/www, "-log", stdout]{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.Replicas.Static | default 0 }}{{ end }}
{{ template "deployment" . }}
