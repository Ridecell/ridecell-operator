{{ define "componentName" }}dispatch{{ end }}
{{ define "componentType" }}dispatch{{ end }}
{{ define "controller" }}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-dispatch{{ end }}
{{ template "verticalPodAutoscaler" . }}
