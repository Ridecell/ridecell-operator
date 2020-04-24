apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-celeryd
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: celeryd
    app.kubernetes.io/instance: {{ .Instance.Name }}-celeryd
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: worker
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    metrics-enabled: {{ .Instance.Spec.Metrics.Celeryd | quote }}
spec:
  replicas: {{ .Instance.Spec.Replicas.Celeryd | default 0 }}
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-celeryd
  template:
    metadata:
      labels:
        app.kubernetes.io/name: celeryd
        app.kubernetes.io/instance: {{ .Instance.Name }}-celeryd
        app.kubernetes.io/version: {{ .Instance.Spec.Version }}
        app.kubernetes.io/component: worker
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
        metrics-enabled: {{ .Instance.Spec.Metrics.Celeryd | quote }}
      annotations:
        summon.ridecell.io/appSecretsHash: {{ .Extra.appSecretsHash }}
        summon.ridecell.io/configHash: {{ .Extra.configHash }}
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              topologyKey: failure-domain.beta.kubernetes.io/zone
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-celeryd
          - weight: 1
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-celeryd
      imagePullSecrets:
      - name: pull-secret
      containers:
      - name: default
        image: us.gcr.io/ridecell-1/summon:{{ .Instance.Spec.Version }}
        imagePullPolicy: Always
        command:
        - python
        - "-m"
        - celery
        - "-A"
        - summon_platform
        - worker
        - "-l"
        - info
        - "--concurrency"
        - {{ .Instance.Spec.Celery.Concurrency | default 30 | quote }}
        - "--pool"
        - {{ .Instance.Spec.Celery.Pool | default "eventlet" | quote }}
        ports:
        - containerPort: 9000
        resources:
          requests:
            memory: 1G
            cpu: 500m
          limits:
            memory: 1.5G
        env:
        - name: SUMMON_COMPONENT
          valueFrom:
            fieldRef:
              fieldPath: metadata.labels['app.kubernetes.io/name']
        {{ if .Instance.Spec.EnableNewRelic }}
        - name: NEW_RELIC_LICENSE_KEY
          valueFrom:
          secretKeyRef:
            name: {{ .Instance.Name }}.newrelic
            key: NEW_RELIC_LICENSE_KEY
        - name: NEW_RELIC_APP_NAME
          value: {{ .Instance.Name }}-summon-platform
        {{ end }}
        {{ if .Instance.Spec.GCPProject }}
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /var/run/secrets/gcp-service-account/google_service_account.json
        {{ end }}
        volumeMounts:
        - name: config-volume
          mountPath: /etc/config
        - name: app-secrets
          mountPath: /etc/secrets
        {{ if .Instance.Spec.EnableNewRelic }}
        - name: newrelic
          mountPath: /home/ubuntu/summon-platform
        {{ end }}
        {{ if .Instance.Spec.GCPProject }}
        - name: gcp-service-account
          mountPath: /var/run/secrets/gcp-service-account
        {{ end }}
        #livenessProbe:
        #  exec:
        #    command:
        #    - bash
        #    - -c
        #    - python -m celery -A summon_platform inspect ping --timeout 60 -d celery@$HOSTNAME
        #  initialDelaySeconds: 300
        #  periodSeconds: 60
        #  failureThreshold: 5
        #  timeoutSeconds: 100
      volumes:
        - name: config-volume
          configMap:
            name: {{ .Instance.Name }}-config
        - name: app-secrets
          secret:
            secretName: {{ .Instance.Name }}.app-secrets
        {{ if .Instance.Spec.EnableNewRelic }}
        - name: newrelic
          secret:
            secretName: {{ .Instance.Name }}.newrelic
        {{ end }}
        {{ if .Instance.Spec.GCPProject }}
        - name: gcp-service-account
          secret:
            secretName: {{ .Instance.Name }}.gcp-credentials
        {{ end }}
