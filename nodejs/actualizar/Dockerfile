FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
<<<<<<< HEAD
RUN npm install --omit=dev
=======
RUN npm ci --omit=dev
>>>>>>> 7a6df03fd809b68140c2da2594444d6196fbaab0
COPY . .
CMD ["npm","start"]
