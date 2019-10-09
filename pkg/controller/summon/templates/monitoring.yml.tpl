apiVersion: monitoring.ridecell.io/v1beta1
kind: Monitor
metadata:
  name: {{ .Instance.Name }}-monitoring
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: monitoring
    app.kubernetes.io/instance: {{ .Instance.Name }}-monitoring
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: monitoring
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  servicename: {{ .Instance.Name }}
  notify:
    slack: [{{ .Instance.Spec.Notifications.SlackChannel | quote }}]
    {{ if .Instance.Spec.Notifications.Pagerdutyteam -}}   
    pagerdutyteam: {{ .Instance.Spec.Notifications.Pagerdutyteam | quote }}
    {{ end -}}
  metricAlertRules:
    - alert: Newrelic error {{ .Instance.Name }}
      expr: newrelic_application_error_rate{appname="{{ .Instance.Name }}-summon-platform"} >= 1
      for: 5m
      labels:
        severity: critical
        servicename: {{ .Instance.Name }}
      annotations:
        summary: Newrelic error % greater than 1 for {{ .Instance.Name }}