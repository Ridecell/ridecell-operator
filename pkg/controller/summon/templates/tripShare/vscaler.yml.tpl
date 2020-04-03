{{ define "componentName" }}tripshare{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "controller" }}
    apiVersion: "apps/v1"
    kind: Deployment
    name: {{ .Instance.Name }}-tripshare{{ end }}
{{ define "updateMode" }}"Off"{{ end }}
{{ template "verticalPodAutoscaler" . }}
