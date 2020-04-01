{{ define "componentName" }}hwaux{{ end }}
{{ define "componentType" }}hwaux{{ end }}
{{ define "controller" }}
  apiVersion: "apps/v1"
  kind: Deployment
  name: {{ .Instance.Name }}-hwaux{{ end }}
{{ define "updateMode" }}"Off"{{ end }}
{{ template "verticalPodAutoscaler" . }}
