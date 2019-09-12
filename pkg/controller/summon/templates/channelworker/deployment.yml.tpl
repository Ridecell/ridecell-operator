{{ define "componentName" }}channelworker{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "command" }}[python, manage.py, runworker, "-v2", "--threads", "2"]{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.Replicas.ChannelWorker | default 0 }}{{ end }}
{{ template "deployment" . }}
