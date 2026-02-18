FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
ARG VITE_API_URL
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
# Railway injeta PORT em tempo de execução; placeholder substituído no startup
RUN echo 'server { listen __PORT__; root /usr/share/nginx/html; index index.html; location / { try_files $uri $uri/ /index.html; } }' > /etc/nginx/conf.d/default.conf
EXPOSE 80
# Substitui __PORT__ pela variável PORT do Railway (padrão 80) e inicia o nginx
CMD ["sh", "-c", "sed -i \"s/__PORT__/${PORT:-80}/g\" /etc/nginx/conf.d/default.conf && exec nginx -g 'daemon off;'"]
