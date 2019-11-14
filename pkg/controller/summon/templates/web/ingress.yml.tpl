{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "ingressPath" }}/{{ end }}
{{ template "ingress" . }}
{{ define "extraAnnotations" }}
    monitor.ridecell.io/healthz-url: "{{ .Instance.Spec.Config.WEB_URL.String }}/healthz"
    monitor.ridecell.io/should-be-probed: "true"
{{ end }}