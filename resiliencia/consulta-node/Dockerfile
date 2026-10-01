FROM node:20-alpine
WORKDIR /app
COPY package*.json ./

RUN npm install --omit=dev
RUN npm ci --omit=dev
COPY . .
CMD ["npm","start"]
