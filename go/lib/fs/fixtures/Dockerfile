FROM alpine:3.20
RUN apk add --update dosfstools mtools sleuthkit
RUN echo "mtools_skip_check=1" >> /etc/mtools.conf
COPY fixtures.sh /opt/fixtures.sh
CMD ["/opt/fixtures.sh"]
