# syntax=docker/dockerfile:1
FROM prom/prometheus:v3.13.2

COPY --chmod=644 prometheus.yml /etc/prometheus/prometheus.yml
