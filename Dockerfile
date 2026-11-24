# syntax=docker/dockerfile:1
# Image autonome : binaire statique Go (stdlib uniquement) sur distroless
# (avec certificats CA pour l'appel HTTPS HelloAsso), déployable tel quel
# sur l'hôte Docker de l'asso ARISE, sans rien installer dessus.

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bds ./cmd/bds

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/bds /app/bds
COPY conf.txt /app/conf.txt
COPY contenu/ /app/contenu/
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/bds", "-conf", "/app/conf.txt"]
