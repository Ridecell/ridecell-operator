{{ define "componentName" }}channelworker{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "controller" }}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-channelworker{{ end }}
{{ define "updateMode" }}"Off"{{ end }}
{{ template "verticalPodAutoscaler" . }}
