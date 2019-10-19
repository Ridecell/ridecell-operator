{{ define "componentName" }}channelworker{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "maxUnavailable" }}{{ if (gt (int .Instance.Spec.Replicas.ChannelWorker) 1) }}10%{{ else }}100%{{ end }}{{ end }}
{{ template "podDisruptionBudget" . }}
