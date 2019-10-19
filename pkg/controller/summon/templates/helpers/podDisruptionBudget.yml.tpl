{{ define "podDisruptionBudget" }}
apiVersion: policy/v1beta1
kind: PodDisruptionBudget
metadata:
  name: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: {{ block "componentName" . }}{{ end }}
    app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  maxUnavailable: 10%
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
{{ end }}
