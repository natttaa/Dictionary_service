# syntax=docker/dockerfile:1
FROM grafana/grafana:13.1.10

USER root
RUN mkdir -p /etc/grafana/dashboards && chmod 755 /etc/grafana/dashboards

COPY --chmod=644 grafana/provisioning/datasources/prometheus.yml /etc/grafana/provisioning/datasources/prometheus.yml
COPY --chmod=644 grafana/provisioning/dashboards/dictionary.yml /etc/grafana/provisioning/dashboards/dictionary.yml
COPY --chmod=644 grafana/dashboards/dictionary-services.json /etc/grafana/dashboards/dictionary-services.json

USER 472:0
