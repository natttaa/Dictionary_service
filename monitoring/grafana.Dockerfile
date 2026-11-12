# syntax=docker/dockerfile:1
FROM grafana/grafana:13.1.10

COPY --chmod=644 grafana/provisioning/datasources/prometheus.yml /etc/grafana/provisioning/datasources/prometheus.yml
COPY --chmod=644 grafana/provisioning/dashboards/dictionary.yml /etc/grafana/provisioning/dashboards/dictionary.yml
COPY --chmod=644 grafana/dashboards/dictionary-services.json /etc/grafana/dashboards/dictionary-services.json
