# This dockerfile is used to build the Switch native binaries
FROM devkitpro/devkita64@sha256:1fc388c3a0d34bd2045a6dadcb1020e069d5f876a187fd705de14b4440c00282

# Install build tools
RUN sudo apt update && sudo apt install -y build-essential

# Install emscripten
RUN git clone https://github.com/emscripten-core/emsdk.git /opt/emsdk \
  && cd /opt/emsdk \
  && ./emsdk install 3.1.65 \
  && ./emsdk activate 3.1.65

COPY ./entrypoint.sh /opt/entrypoint.sh
ENTRYPOINT ["/opt/entrypoint.sh"]
