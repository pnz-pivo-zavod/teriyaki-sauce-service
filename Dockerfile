FROM golang:1.27 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY api ./api
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/teriyaki-sauce-service ./cmd

# static: бинарник без CGO, миграции и SQL вшиты через embed — больше ничего не нужно.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/teriyaki-sauce-service /teriyaki-sauce-service

EXPOSE 8080
ENTRYPOINT ["/teriyaki-sauce-service"]
