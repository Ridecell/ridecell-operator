apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Instance.Name }}-redis
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: redis
    app.kubernetes.io/instance: {{ .Instance.Name }}-redis
    app.kubernetes.io/component: database
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/instance: {{ .Instance.Name }}-redis
  template:
    metadata:
      labels:
        app.kubernetes.io/name: redis
        app.kubernetes.io/instance: {{ .Instance.Name }}-redis
        app.kubernetes.io/component: database
        app.kubernetes.io/part-of: {{ .Instance.Name }}
        app.kubernetes.io/managed-by: summon-operator
    spec:
      containers:
      - name: default
        image: redis:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 6379
        args:
        - "--appendonly"
        - "yes"
        volumeMounts:
        - name: redis-persist
          mountPath: /data
        resources:
          requests:
            memory: 100M
            cpu: 100m
          limits:
            memory: 1G
        readinessProbe:
          exec:
            command:
            - sh
            - -c
            - "redis-cli -h $(hostname) ping"
          initialDelaySeconds: 10
          periodSeconds: 5
        livenessProbe:
          exec:
            command:
            - sh
            - -c
            - "redis-cli -h $(hostname) ping"
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: redis-persist
        persistentVolumeClaim:
          claimName: {{ .Instance.Name }}-redis
