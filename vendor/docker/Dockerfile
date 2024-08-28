# This dockerfile is used to build the Switch native binaries
FROM devkitpro/devkita64@sha256:241527399a65d16aaffb9f79b5453f03c0a068413c48b35277e503bb5525e2c0

# Install emscripten
RUN git clone https://github.com/emscripten-core/emsdk.git /opt/emsdk \
  && cd /opt/emsdk \
  && ./emsdk install 3.1.65 \
  && ./emsdk activate 3.1.65

COPY ./entrypoint.sh /opt/entrypoint.sh
ENTRYPOINT ["/opt/entrypoint.sh"]
