apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-businessportal
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: businessportal
    app.kubernetes.io/instance: {{ .Instance.Name }}-businessportal
    app.kubernetes.io/version: {{ .Instance.Spec.BusinessPortal.Version | quote }}
    app.kubernetes.io/component: web
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    metrics-enabled: "false"
spec:
  replicas: {{ .Instance.Spec.Replicas.BusinessPortal }}
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-businessportal
  template:
    metadata:
      labels:
        app.kubernetes.io/name: businessportal
        app.kubernetes.io/instance: {{ .Instance.Name }}-businessportal
        app.kubernetes.io/version: {{ .Instance.Spec.BusinessPortal.Version | quote }}
        app.kubernetes.io/component: web
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
        metrics-enabled: "false"
    spec:
      serviceAccountName: {{ .Instance.Name }}
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app.kubernetes.io/instance: {{ .Instance.Name }}-businessportal
            topologyKey: "kubernetes.io/hostname"
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 1
            podAffinityTerm:
              topologyKey: "failure-domain.beta.kubernetes.io/zone"
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-businessportal
      imagePullSecrets:
      - name: pull-secret
      containers:
      - name: default
        image: "us.gcr.io/ridecell-1/comp-business-portal:{{ .Instance.Spec.BusinessPortal.Version }}"
        ports:
        - containerPort: 8000
        resources:
          requests:
            memory: 60M
            cpu: 5m
          limits:
            memory: 100M
        env:
        - name: SUMMON_COMPONENT
          valueFrom:
            fieldRef:
              fieldPath: metadata.labels['app.kubernetes.io/name']
        readinessProbe:
          httpGet:
            path: /
            port: 8000
          periodSeconds: 2
        livenessProbe:
          httpGet:
            path: /
            port: 8000
          initialDelaySeconds: 60
