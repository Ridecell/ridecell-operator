{{ define "componentName" }}static{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "controller" }}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-static{{ end }}
{{ template "verticalPodAutoscaler" . }}
