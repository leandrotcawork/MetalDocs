FROM node:20-alpine AS builder
# Build context is the repo root so shared/ is accessible alongside frontend/.
# canvas (dep of pdf-img-convert devDep) requires native compilation on musl Linux.
# Install build tools so it compiles from source; only affects the builder stage.
RUN apk add --no-cache python3 make g++ cairo-dev pango-dev jpeg-dev giflib-dev librsvg-dev

WORKDIR /app
# Install dependencies from the web app subdirectory so node_modules land next to package.json.
COPY frontend/apps/web/package.json frontend/apps/web/package-lock.json ./frontend/apps/web/
RUN cd frontend/apps/web && npm ci

# Copy web app source and the shared schemas that pipeline.ts / adapter.test.ts import
# via relative paths (e.g. ../../../../../../../../../shared/schemas/canonicalize).
COPY frontend/apps/web/ ./frontend/apps/web/
COPY shared/ ./shared/

RUN cd frontend/apps/web && npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/frontend/apps/web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/frontend/apps/web/dist /usr/share/nginx/html
EXPOSE 4173
