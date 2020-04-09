{{ define "componentName" }}businessportal{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "controller" }}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-businessportal{{ end }}
{{ template "verticalPodAutoscaler" . }}
