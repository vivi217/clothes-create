FROM m.daocloud.io/docker.io/library/node:22-bookworm-slim

WORKDIR /app

COPY package.json tsconfig.json vite.config.ts tailwind.config.js postcss.config.js index.html ./
COPY src ./src
COPY electron ./electron

RUN npm install

EXPOSE 3000

CMD ["npm", "run", "dev"]