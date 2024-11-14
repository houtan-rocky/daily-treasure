FROM envoyproxy/envoy:v1.27.0

COPY envoy.yaml /etc/envoy/envoy.yaml

CMD ["/usr/local/bin/envoy", "-c", "/etc/envoy/envoy.yaml", "-l", "info"]