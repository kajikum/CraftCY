FROM java:openjdk-8-jre

MAINTAINER zak1ck

WORKDIR /minecraft
COPY entrypoint.sh /minecraft/entrypoint.sh
RUN useradd minecraft && chown minecraft entrypoint.sh && chmod +x entrypoint.sh

USER minecraft
VOLUME ["/minecraft/data"]
EXPOSE 25565
ENTRYPOINT ["./entrypoint.sh"]
