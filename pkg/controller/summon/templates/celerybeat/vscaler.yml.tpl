{{ define "componentName" }}celerybeat{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "controller" }}
  apiVersion: "apps/v1"
  kind: StatefulSet
  name: {{ .Instance.Name }}-celerybeat{{ end }}
{{ define "updateMode" }}"Off"{{ end }}
{{ template "verticalPodAutoscaler" . }}
