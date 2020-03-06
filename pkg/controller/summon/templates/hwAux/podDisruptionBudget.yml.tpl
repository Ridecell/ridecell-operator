{{ define "componentName" }}hxaux{{ end }}
{{ define "componentType" }}hxaux{{ end }}
{{ define "maxUnavailable" }}{{ if (gt (int .Instance.Spec.Replicas.HwAux) 1) }}10%{{ else }}100%{{ end }}{{ end }}
{{ template "podDisruptionBudget" . }}
