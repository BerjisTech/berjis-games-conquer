# Build stage
FROM node:20-alpine AS build
WORKDIR /workspace
COPY clients/angular-auth ./clients/angular-auth
WORKDIR /workspace/games/conquer
COPY games/conquer/package.json ./package.json
COPY games/conquer/yarn.lock ./yarn.lock
RUN corepack enable \
 && if [ -f yarn.lock ]; then yarn install --immutable; else yarn install; fi \
 && ln -s /workspace/games/conquer/node_modules /workspace/node_modules
COPY games/conquer/ .
RUN yarn build

# Runtime stage
FROM nginx:1.27-alpine
COPY --from=build /workspace/games/conquer/dist/conquer/browser /usr/share/nginx/html
COPY games/conquer/nginx.conf /etc/nginx/conf.d/default.conf
COPY games/conquer/env.default.js /usr/share/nginx/html/env.js
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
