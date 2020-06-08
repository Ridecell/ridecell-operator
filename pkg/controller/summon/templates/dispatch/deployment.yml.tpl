apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-dispatch
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: dispatch
    app.kubernetes.io/instance: {{ .Instance.Name }}-dispatch
    app.kubernetes.io/version: {{ .Instance.Spec.Dispatch.Version | quote }}
    app.kubernetes.io/component: dispatch
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    metrics-enabled: "false"
spec:
  replicas: {{ .Instance.Spec.Replicas.Dispatch }}
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-dispatch
  template:
    metadata:
      labels:
        app.kubernetes.io/name: dispatch
        app.kubernetes.io/instance: {{ .Instance.Name }}-dispatch
        app.kubernetes.io/version: {{ .Instance.Spec.Dispatch.Version | quote }}
        app.kubernetes.io/component: dispatch
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
        metrics-enabled: "false"
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
                  app.kubernetes.io/instance: {{ .Instance.Name }}-dispatch
          - weight: 1
            podAffinityTerm:
              topologyKey: kubernetes.io/hostname
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-dispatch
      imagePullSecrets:
      - name: pull-secret
      containers:
      - name: default
        image: "us.gcr.io/ridecell-1/comp-dispatch:{{ .Instance.Spec.Dispatch.Version }}"
        ports:
        - containerPort: 8000
        resources: {{ .Instance.Spec.Dispatch.Resources | toJson }}
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
        - name: dispatch-config
          mountPath: /etc/config
        {{ if .Instance.Spec.EnableNewRelic }}
        - name: newrelic
          mountPath: /home/ubuntu/summon-platform
        {{ end }}
        {{ if .Instance.Spec.GCPProject }}
        - name: gcp-service-account
          mountPath: /var/run/secrets/gcp-service-account
        {{ end }}
      volumes:
        - name: dispatch-config
          secret:
            secretName: {{ .Instance.Name }}.dispatch
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
