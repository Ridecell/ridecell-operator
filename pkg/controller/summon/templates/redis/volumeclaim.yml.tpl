apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ .Instance.Name }}-redis
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: redis
    app.kubernetes.io/instance: {{ .Instance.Name }}-redis
    app.kubernetes.io/component: database
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
			# If this value changes you will need to update defaults.go
      storage: 10Gi
  storageClassName: gp2
