# Build stage
FROM node:20-alpine AS build
WORKDIR /app
COPY package.json yarn.lock* ./
RUN corepack enable \
 && if [ -f yarn.lock ]; then yarn install --immutable; else yarn install; fi
COPY . .
RUN yarn build

# Runtime stage
FROM nginx:1.27-alpine
COPY --from=build /app/dist/conquer/browser /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY env.default.js /usr/share/nginx/html/env.js
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
