apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-customerportal
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: customerportal
    app.kubernetes.io/instance: {{ .Instance.Name }}-customerportal
    app.kubernetes.io/version: {{ .Instance.Spec.CustomerPortal.Version | quote }}
    app.kubernetes.io/component: web
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    metrics-enabled: "false"
spec:
  replicas: {{ .Instance.Spec.Replicas.CustomerPortal }}
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-customerportal
  template:
    metadata:
      labels:
        app.kubernetes.io/name: customerportal
        app.kubernetes.io/instance: {{ .Instance.Name }}-customerportal
        app.kubernetes.io/version: {{ .Instance.Spec.CustomerPortal.Version | quote }}
        app.kubernetes.io/component: web
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
        metrics-enabled: "false"
      {{ if .Instance.Spec.UseIamRole }}
      annotations:
        iam.amazonaws.com/role: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
      {{ end }}
    spec:
      {{ if .Instance.Spec.UseIamRole }}
      serviceAccountName: {{ .Instance.Name }}
      {{ end }}
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              topologyKey: failure-domain.beta.kubernetes.io/zone
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-customerportal
          - weight: 1
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-customerportal
      imagePullSecrets:
      - name: pull-secret
      containers:
      - name: default
        image: "us.gcr.io/ridecell-1/comp-customer-portal:{{ .Instance.Spec.CustomerPortal.Version }}"
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
        - name: config-volume
          mountPath: /etc/config
        - name: app-secrets
          mountPath: /etc/secrets
      volumes:
        - name: config-volume
          configMap:
            name: {{ .Instance.Name }}-config
        - name: app-secrets
          secret:
            secretName: {{ .Instance.Name }}.app-secrets
