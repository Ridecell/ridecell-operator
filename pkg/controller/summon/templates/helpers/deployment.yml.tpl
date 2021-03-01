{{ define "deployment" }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: {{ block "componentName" . }}{{ end }}
    app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    metrics-enabled: {{ block "metricsEnabled" . }}{{ end }}
spec:
  replicas: {{ block "replicas" . }}1{{ end }}
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ block "componentName" . }}{{ end }}
        app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
        app.kubernetes.io/version: {{ .Instance.Spec.Version }}
        app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
        metrics-enabled: {{ block "metricsEnabled" . }}{{ end }}
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
          requiredDuringSchedulingIgnoredDuringExecution:
          - topologyKey: kubernetes.io/hostname
            labelSelector:
              matchLabels:
                app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 1
            podAffinityTerm:
              topologyKey: failure-domain.beta.kubernetes.io/zone
              labelSelector:
                matchLabels:
                  app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
      imagePullSecrets:
      - name: pull-secret
      containers:
      - name: default
        image: us.gcr.io/ridecell-1/summon:{{ .Instance.Spec.Version }}
        imagePullPolicy: Always
        command: {{ block "command" . }}[]{{ end }}
        ports: {{ block "deploymentPorts" . }}[{containerPort: 8000}]{{ end }}
        resources: {{ block "resources" . }}{}{{ end }}
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
        - name: saml
          mountPath: /etc/saml
        {{ if .Instance.Spec.EnableNewRelic }}
        - name: newrelic
          mountPath: /home/ubuntu/summon-platform
        {{ end }}
        {{ if .Instance.Spec.GCPProject }}
        - name: gcp-service-account
          mountPath: /var/run/secrets/gcp-service-account
        {{ end }}
        {{ block "containerExtra" . }}{{ end }}
      volumes:
        - name: config-volume
          configMap:
            name: {{ .Instance.Name }}-config
        - name: app-secrets
          secret:
            secretName: {{ .Instance.Name }}.app-secrets
        - name: saml
          secret:
            secretName: {{ .Instance.Name }}.saml
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
{{ end }}
