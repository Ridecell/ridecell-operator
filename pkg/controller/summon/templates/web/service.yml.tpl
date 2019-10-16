{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "extraAnnotations" }} 
    monitor.ridecell.io/healthz-path: healthz
    monitor.ridecell.io/should-be-probed: "true"
{{ end }}
{{ template "service" . }}
