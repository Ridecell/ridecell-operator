{{ define "componentName" }}celeryd{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "controller" }}
  apiVersion: "apps/v1"
  kind: Deployment
  name: {{ .Instance.Name }}-celeryd{{ end }}
{{ define "updateMode" }}"Off"{{ end }}
{{ template "verticalPodAutoscaler" . }}
