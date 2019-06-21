apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  labels:
    role: prometheus-infra-rules
  name: {{ .Instance.Name| quote }}
  namespace: {{ .Instance.Namespace | quote }}
spec:
  groups: 
  - name: {{ .Instance.Name | printf "%srules" | quote }} 
    rules: {{ .Instance.Spec.MetricAlertRules | toJson }}
