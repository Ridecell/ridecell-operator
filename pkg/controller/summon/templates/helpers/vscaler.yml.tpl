{{ define "verticalPodAutoscaler" }}
apiVersion: autoscaling.k8s.io/v1beta2
kind: VerticalPodAutoscaler
metadata:
  name: {{ .Instance.Name }}-{{block "componentName" . }}{{ end }}
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: {{ block "componentName" .}}{{ end }}
    app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    {{/* creator and source needed for goldilocks dashboard to pick vpa up */ -}}
    creator: "Fairwinds"
    source: "goldilocks"
spec:
  TargetRef: {{ block "controller" . }}{{ end }}
  UpdatePolicy:
    UpdateMode: {{ block "updateMode" . }}{{ end }}
  {{/* Leaving ResourcePolicy blank so autoscaler can compute recommended */ -}}
{{ end }}
