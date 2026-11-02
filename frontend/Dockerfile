FROM node:22-alpine AS build
WORKDIR /app
# Fixed-token MVP: value is inlined into the static bundle at build time.
# Never pass a model/provider key here; only AGENTHUB_API_TOKEN.
ARG VITE_AGENTHUB_API_TOKEN=
ENV VITE_AGENTHUB_API_TOKEN=$VITE_AGENTHUB_API_TOKEN
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS runtime
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 3000
CMD ["nginx", "-g", "daemon off;", "-c", "/etc/nginx/nginx.conf"]
