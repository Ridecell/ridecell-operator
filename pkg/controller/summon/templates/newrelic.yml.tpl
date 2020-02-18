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
    error_collector.ignore_errors = celery.exceptions:Retry
    error_collector.ignore_status_codes = 100-102 200-208 226 300-308 400-499
    browser_monitoring.auto_instrument = false
  NEW_RELIC_LICENSE_KEY: {{ env "NEW_RELIC_LICENSE_KEY" }}
