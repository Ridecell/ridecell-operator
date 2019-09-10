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
      annotations:
        summon.ridecell.io/appSecretsHash: {{ .Extra.appSecretsHash }}
        summon.ridecell.io/configHash: {{ .Extra.configHash }}
    spec:
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
        {{ if .Instance.Spec.Celery.Concurrency }}
        - "--concurrency"
        - {{ .Instance.Spec.Celery.Concurrency | quote }}
        {{ end }}
        - "--pool"
        - {{ .Instance.Spec.Celery.Pool | default "prefork" }}
        ports:
        - containerPort: 8000
        resources:
          requests:
            memory: 512M
            cpu: 500m
          limits:
            memory: 3G
            cpu: 1000m
        {{ if .Instance.Spec.EnableNewRelic }}
        env:
        - name: NEW_RELIC_LICENSE_KEY
          valueFrom:
          secretKeyRef:
            name: {{ .Instance.Name }}.newrelic
            key: NEW_RELIC_LICENSE_KEY
        - name: NEW_RELIC_APP_NAME
          value: {{ .Instance.Name }}-summon-platform
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
        livenessProbe:
          exec:
            command:
            - bash
            - -c
            - python -m celery -A summon_platform inspect ping -d celery@$HOSTNAME
          initialDelaySeconds: 10
          periodSeconds: 5
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

