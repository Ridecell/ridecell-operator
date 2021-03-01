apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-celeryredbeat
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: celeryredbeat
    app.kubernetes.io/instance: {{ .Instance.Name }}-celeryredbeat
    app.kubernetes.io/version: {{ .Instance.Spec.Version | quote }}
    app.kubernetes.io/component: worker
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  replicas: {{ .Instance.Spec.Replicas.CeleryRedBeat }}
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-celeryredbeat
  template:
    metadata:
      labels:
        app.kubernetes.io/name: celeryredbeat
        app.kubernetes.io/instance: {{ .Instance.Name }}-celeryredbeat
        app.kubernetes.io/version: {{ .Instance.Spec.Version }}
        app.kubernetes.io/component: worker
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
      annotations:
        summon.ridecell.io/appSecretsHash: {{ .Extra.appSecretsHash }}
        summon.ridecell.io/configHash: {{ .Extra.configHash }}
    spec:
      {{ if .Instance.Spec.UseIamRole }}
      serviceAccountName: {{ .Instance.Name }}
      {{ end }}
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app.kubernetes.io/instance: {{ .Instance.Name }}-celeryredbeat
            topologyKey: "kubernetes.io/hostname"
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 1
            podAffinityTerm:
              topologyKey: "failure-domain.beta.kubernetes.io/zone"
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-celeryredbeat
      imagePullSecrets:
      - name: pull-secret
      containers:
      - name: default
        image: us.gcr.io/ridecell-1/summon:{{ .Instance.Spec.Version }}
        command:
        - /bin/sh
        - -c
        {{ if .Extra.debug }}
        - python -m celery -A summon_platform beat -l debug -S redbeat.RedBeatScheduler --pidfile=
        {{ else }}
        - python -m celery -A summon_platform beat -l info -S redbeat.RedBeatScheduler --pidfile=
        {{ end }}
        resources:
          requests:
            memory: 260M
            cpu: 5m
          limits:
            memory: 500M
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
        volumeMounts:
        - name: config-volume
          mountPath: /etc/config
        - name: app-secrets
          mountPath: /etc/secrets
        {{ if .Instance.Spec.EnableNewRelic }}
        - name: newrelic
          mountPath: /home/ubuntu/summon-platform
        {{ end }}
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
