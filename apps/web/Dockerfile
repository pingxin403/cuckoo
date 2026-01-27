# Multi-stage build for React + Vite application
# NOTE: This Dockerfile should be built from the repository root directory
# Example: docker build -f apps/web/Dockerfile -t web:latest .

# Stage 1: Build the application
FROM node:18-alpine AS builder

WORKDIR /app

# Copy centralized proto generated code
COPY api/gen/typescript ./api/gen/typescript

# Copy web app package files
COPY apps/web/package*.json ./apps/web/

WORKDIR /app/apps/web

# Install dependencies
RUN npm ci

# Copy web app source code
COPY apps/web/ ./

# Build the application
RUN npm run build

# Stage 2: Serve with nginx
FROM nginx:1.25-alpine

# Copy built assets from builder stage
COPY --from=builder /app/apps/web/dist /usr/share/nginx/html

# Copy nginx configuration
COPY apps/web/nginx.conf /etc/nginx/conf.d/default.conf

# Expose port 80
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
