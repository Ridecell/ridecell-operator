{{ define "componentName" }}daphne{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "controller" }}
  apiVersion: "apps/v1"
  kind: Deployment
  name: {{ .Instance.Name }}-daphne{{ end }}
{{ define "updateMode" }}"Off"{{ end }}
{{ template "verticalPodAutoscaler" . }}
