{{ define "componentName" }}redis{{ end }}
{{ define "componentType" }}redis{{ end }}
{{ define "controller" }}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-redis{{ end }}
{{ define "updateMode" }}"Off"{{ end }}
{{ template "verticalPodAutoscaler" . }}
