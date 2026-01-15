# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package.json ./

# Clean install to ensure correct versions
RUN rm -rf node_modules package-lock.json && \
    npm cache clean --force && \
    npm install next@16.1.1 react@19.0.0 react-dom@19.0.0 --legacy-peer-deps && \
    npm install --legacy-peer-deps

# Copy source code
COPY . .

# Build the application (use webpack instead of turbopack for compatibility)
RUN npm run build -- --webpack

# Production stage
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

# Create non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy built application
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT 3000
ENV HOSTNAME "0.0.0.0"

CMD ["node", "server.js"]

