# Build stage
FROM node:22-alpine as build-stage
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy built assets from build-stage to Nginx serve directory
COPY --from=build-stage /app/dist /usr/share/nginx/html

# The configuration will be injected via volume mount in docker-compose
EXPOSE 80
# EXPOSE 443

# Command is overridden in docker-compose.yml to enable the reload loop
CMD ["nginx", "-g", "daemon off;"]
