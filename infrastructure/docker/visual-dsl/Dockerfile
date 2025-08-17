# Visual DSL - Node.js development server
FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Install serve to run the built app
RUN npm install -g serve

EXPOSE 3000

# Serve the built application (serve binds to 0.0.0.0 by default)
CMD ["serve", "-s", "dist", "-p", "3000"]
