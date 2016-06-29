FROM scratch
ADD web-proxy /web-proxy
ENTRYPOINT [ "/web-proxy" ]
