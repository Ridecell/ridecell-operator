{{ define "componentName" }}channelworker{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ template "podDisruptionBudget" . }}
