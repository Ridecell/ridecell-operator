apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-tripshare
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: tripshare
    app.kubernetes.io/instance: {{ .Instance.Name }}-tripshare
    app.kubernetes.io/version: {{ .Instance.Spec.TripShare.Version | quote }}
    app.kubernetes.io/component: web
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    metrics-enabled: "false"
spec:
  replicas: {{ .Instance.Spec.Replicas.TripShare }}
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-tripshare
  template:
    metadata:
      labels:
        app.kubernetes.io/name: tripshare
        app.kubernetes.io/instance: {{ .Instance.Name }}-tripshare
        app.kubernetes.io/version: {{ .Instance.Spec.TripShare.Version | quote }}
        app.kubernetes.io/component: web
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
        metrics-enabled: "false"
      annotations:
        summon.ridecell.io/appSecretsHash: {{ .Extra.appSecretsHash }}
        summon.ridecell.io/configHash: {{ .Extra.configHash }}
        iam.amazonaws.com/role: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              topologyKey: failure-domain.beta.kubernetes.io/zone
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-tripshare
          - weight: 1
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-tripshare
      containers:
      - name: default
        image: "us.gcr.io/ridecell-1/comp-trip-share:{{ .Instance.Spec.TripShare.Version }}"
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
        volumeMounts:
        - name: config
          mountPath: /app/_site/trip_share/config.json
          subPath: config.json
      volumes:
      - name: config
        secret:
          secretName: {{ .Instance.Name }}.tripshare
