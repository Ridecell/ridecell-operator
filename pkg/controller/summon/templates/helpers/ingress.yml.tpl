{{ define "ingress" }}
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: {{ block "componentName" . }}{{ end }}
    app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
  annotations:
    kubernetes.io/ingress.class: traefik
    kubernetes.io/tls-acme: "true"
    certmanager.k8s.io/cluster-issuer: letsencrypt-prod
spec:
  rules:
  - host: {{ .Instance.Spec.Hostname }}
    http:
      paths:
      - path: {{ block "ingressPath" . }}{{ end }}
        backend:
          serviceName: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
          servicePort: 8000
  {{range .Instance.Spec.Aliases}}- host: {{.}}
    http:
      paths:
      - path: {{ block "ingressPath" $ }}{{ end }}
        backend:
          serviceName: {{ $.Instance.Name }}-{{ block "componentName" $ }}{{ end }}
          servicePort: 8000{{end}}
  tls:
  - secretName: {{ .Instance.Name }}-tls
    hosts:
    - {{ .Instance.Spec.Hostname }}
    {{range .Instance.Spec.Aliases}}- {{.}}
    {{ end }}
{{ end }}