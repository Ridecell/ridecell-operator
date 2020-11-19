kind: Service
apiVersion: v1
metadata:
  name: {{ .Instance.Name }}-celeryredbeat
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: celeryredbeat
    app.kubernetes.io/instance: {{ .Instance.Name }}-celeryredbeat
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: worker
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  selector:
    app.kubernetes.io/instance: {{ .Instance.Name }}-celeryredbeat
  clusterIP: None
