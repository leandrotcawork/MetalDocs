FROM node:20.11-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
COPY apps/docgen-v2/package.json ./apps/docgen-v2/
RUN npm ci --workspace @metaldocs/docgen-v2 --include-workspace-root
COPY apps/docgen-v2 ./apps/docgen-v2
RUN npm run build --workspace @metaldocs/docgen-v2

FROM node:20.11-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/apps/docgen-v2/dist ./dist
COPY --from=build /app/apps/docgen-v2/package.json ./package.json
USER node
EXPOSE 3100
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:3100/health || exit 1
CMD ["node", "dist/index.js"]
