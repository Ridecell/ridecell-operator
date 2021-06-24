{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    app.kubernetes.io/name: {{ block "componentName" . }}{{ end }}
    app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    kubernetes.io/ingress.class: traefik
    traefik.ingress.kubernetes.io/router.middlewares: traefik-traefik-forward-auth@kubernetescrd
  name: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}-protected
  namespace: {{ .Instance.Namespace }}
spec:
  rules:
  - host: {{ .Instance.Spec.Hostname }}
    http:
      paths:
      - backend:
          serviceName: {{ $.Instance.Name }}-{{ block "componentName" $ }}{{ end }}
          servicePort: 8000
        path: /admin
        pathType: Prefix
  - host: {{ .Instance.Name }}.ridecell.io
    http:
      paths:
      - backend:
          serviceName: {{ $.Instance.Name }}-{{ block "componentName" $ }}{{ end }}
          servicePort: 8000
        path: /admin
        pathType: Prefix
  {{- range .Instance.Spec.Aliases }}
  - host: {{.}}
    http:
      paths:
      - path: /admin
        pathType: Prefix
        backend:
          serviceName: {{ $.Instance.Name }}-{{ block "componentName" $ }}{{ end }}
          servicePort: 8000
  {{- end }}
  tls:
  - secretName: {{ .Instance.Name }}-tls
    hosts:
    - {{ .Instance.Spec.Hostname }}
    {{- range .Instance.Spec.Aliases }}
    - {{.}}
    {{- end }}
