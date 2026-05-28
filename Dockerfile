# ─── Stage 1: Frontend build ───
FROM node:22-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package.json ./
RUN npm install
COPY frontend/ .
RUN npm run build

# ─── Stage 2: Backend build ───
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
COPY --from=frontend-builder /app/build ./frontend/build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cifras .

# ─── Stage 3: Final image (distroless / scratch) ───
FROM scratch
WORKDIR /app
COPY --from=backend-builder /app/cifras .
COPY --from=backend-builder /app/frontend/build ./frontend/build
EXPOSE 8080
ENV PORT=8080
CMD ["./cifras"]