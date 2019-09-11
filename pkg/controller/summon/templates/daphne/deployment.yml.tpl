{{ define "componentName" }}daphne{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "command" }}[daphne, "-b", "0.0.0.0", "summon_platform.asgi:channel_layer"]{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.Replicas.Daphne | default 0 }}{{ end }}
{{ template "deployment" . }}
