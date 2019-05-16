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
  replicas: {{ .Instance.Spec.WorkerReplicas }}
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
        command: [python, "-m", celery, "-A", summon_platform, worker, "-l", info]
        ports:
        - containerPort: 8000
        resources:
          requests:
            memory: 512M
            cpu: 500m
          limits:
            memory: 3G
            cpu: 1000m
        volumeMounts:
        - name: config-volume
          mountPath: /etc/config
        - name: app-secrets
          mountPath: /etc/secrets
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
