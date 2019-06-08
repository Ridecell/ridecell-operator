apiVersion: v1
kind: Secret
metadata:
  name: {{ .Instance.Name }}.newrelic
  namespace: {{ .Instance.Namespace }}
stringData:
  newrelic.ini: |
    [newrelic]
    license_key = {{ env "NEW_RELIC_LICENSE_KEY" }}
    app_name = {{ .Instance.Name }}-summon-platform
  NEW_RELIC_LICENSE_KEY: {{ env "NEW_RELIC_LICENSE_KEY" }}
