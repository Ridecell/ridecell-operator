kind: MigrationJob
apiVersion: db.ridecell.io/v1beta1
metadata:
 name: {{ .Instance.Name }}
 namespace: {{ .Instance.Namespace }}
 labels:
   app.kubernetes.io/name: migrations
   app.kubernetes.io/instance: {{ .Instance.Name }}-migrations
   app.kubernetes.io/version: {{ .Instance.Spec.Version }}
   app.kubernetes.io/component: migration
   app.kubernetes.io/part-of: {{ .Instance.Name }}
   app.kubernetes.io/managed-by: summon-operator
spec:
 template:
   metadata:
     labels:
       app.kubernetes.io/name: migrations
       app.kubernetes.io/instance: {{ .Instance.Name }}-migrations
       app.kubernetes.io/version: {{ .Instance.Spec.Version }}
       app.kubernetes.io/component: migration
       app.kubernetes.io/part-of: {{ .Instance.Name }}
       app.kubernetes.io/managed-by: summon-operator
   spec:
     restartPolicy: Never
     imagePullSecrets:
     - name: pull-secret
     containers:
     - name: default
       image: us.gcr.io/ridecell-1/summon:{{ .Instance.Spec.Version }}
       imagePullPolicy: Always
       command:
       - sh
       - "-c"
       {{- if ne .Extra.presignedUrl "" }}
       - python manage.py migrate -v3 && python manage.py loadflavor {{ .Extra.presignedUrl | squote }} --silent
       {{- else }}
       - {{ if and (not .Instance.Spec.NoCore1540Fixup) (ne .Instance.Status.MigrateVersion "") }}if [ -f common/management/commands/core_1540_pre_migrate.py ]; then python manage.py core_1540_pre_migrate; fi && {{ end }}python manage.py migrate -v3
       {{- end }}
       resources:
         requests:
           memory: 1.5G
           cpu: 500m
         limits:
           memory: 2.5G
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
