{{ define "componentName" }}static{{ end }}
{{ define "componentType" }}web{{ end }}
{{ template "podDisruptionBudget" . }}
