FROM node:22-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

FROM alpine:3.21
COPY --from=builder /app/dist /dist
CMD ["cp", "-r", "/dist/.", "/srv/"]