FROM node:20-bookworm-slim
WORKDIR /app
COPY package.json ./
COPY web ./web
COPY server ./server
COPY agent ./agent
COPY data ./data
RUN mkdir -p /app/uploads
ENV NODE_ENV=production
ENV PORT=10000
EXPOSE 10000
CMD ["node","server/server.js"]
