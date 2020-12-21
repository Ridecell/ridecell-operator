apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-kafkaconsumer
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: kafkaconsumer
    app.kubernetes.io/instance: {{ .Instance.Name }}-kafkaconsumer
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: worker
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    metrics-enabled: "{{ .Instance.Spec.Metrics.kafkaconsumer | default "false" }}"
spec:
  replicas: {{ .Instance.Spec.Replicas.kafkaconsumer | default 0 }}
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-kafkaconsumer
  template:
    metadata:
      labels:
        app.kubernetes.io/name: kafkaconsumer
        app.kubernetes.io/instance: {{ .Instance.Name }}-kafkaconsumer
        app.kubernetes.io/version: {{ .Instance.Spec.Version }}
        app.kubernetes.io/component: worker
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
        metrics-enabled: "{{ .Instance.Spec.Metrics.kafkaconsumer | default "false" }}"
      annotations:
        summon.ridecell.io/appSecretsHash: {{ .Extra.appSecretsHash }}
        summon.ridecell.io/configHash: {{ .Extra.configHash }}
        {{ if .Instance.Spec.UseIamRole }}
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
                  app.kubernetes.io/instance: {{ .Instance.Name }}-kafkaconsumer
          - weight: 1
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-kafkaconsumer
      imagePullSecrets:
      - name: pull-secret
      containers:
      - name: default
        image: us.gcr.io/ridecell-1/summon:{{ .Instance.Spec.Version }}
        imagePullPolicy: Always
        command:
        - python
        - /src/manage.py
        - run_kafka_consumer
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