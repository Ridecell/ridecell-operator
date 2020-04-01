{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "controller" }}
  apiVersion: "apps/v1"
  kind: Deployment
  name: {{ .Instance.Name }}-web{{ end }}
{{ define "updateMode" }}"Off"{{ end }}
{{ template "verticalPodAutoscaler" . }}
