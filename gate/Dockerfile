#golang:1.18.0-alpine3.15
FROM ubuntu

#WORKDIR /src
COPY bin/gate /gate
#RUN chmod +x /gate

EXPOSE 8080
ENTRYPOINT [ "/gate" ]