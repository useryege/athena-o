FROM node:22-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts

COPY tsconfig.json tsconfig.build.json ./
COPY cmd/ ./cmd/
COPY src/ ./src/

RUN npm run build

# --- production stage ---
FROM node:22-alpine AS runner

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --omit=dev --ignore-scripts

COPY --from=builder /app/dist ./dist

EXPOSE 50051

ENV NODE_ENV=production

CMD ["node", "dist/cmd/server/main.js"]
