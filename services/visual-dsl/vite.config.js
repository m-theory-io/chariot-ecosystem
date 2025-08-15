import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";
import fs from "fs";

// Check for SSL certificates and determine if HTTPS should be enabled
const getHttpsConfig = () => {
  const certPath = './certs/server.crt';
  const keyPath = './certs/server.key';
  
  // Force HTTPS if environment variable is set
  if (process.env.HTTPS === 'true') {
    if (fs.existsSync(certPath) && fs.existsSync(keyPath)) {
      console.log('🔐 HTTPS enabled with custom certificates');
      return {
        key: fs.readFileSync(keyPath),
        cert: fs.readFileSync(certPath),
      };
    } else {
      console.log('🔐 HTTPS enabled with Vite default certificates');
      return true; // Let Vite generate its own certificates
    }
  }
  
  // Auto-enable HTTPS if certificates exist
  if (fs.existsSync(certPath) && fs.existsSync(keyPath)) {
    console.log('🔐 HTTPS auto-enabled (certificates found)');
    return {
      key: fs.readFileSync(keyPath),
      cert: fs.readFileSync(certPath),
    };
  }
  
  return false;
};

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    host: '0.0.0.0', // Bind to all network interfaces
    port: 5174,      // Set a consistent port
    https: getHttpsConfig(),
    strictPort: true, // Don't try another port if 5174 is busy
    // Firefox CSP compatibility
    headers: {
      'Content-Security-Policy': [
        "default-src 'self'",
        "script-src 'self' 'unsafe-inline' 'unsafe-eval' resource: chrome: moz-extension: ws: wss:",
        "style-src 'self' 'unsafe-inline'",
        "img-src 'self' data: blob:",
        "font-src 'self' data:",
        "connect-src 'self' ws: wss: http: https:",
        "frame-src 'self'",
        "worker-src 'self' blob:",
        "object-src 'none'",
        "base-uri 'self'"
      ].join('; '),
      'X-Content-Type-Options': 'nosniff',
      'X-Frame-Options': 'SAMEORIGIN',
    },
  },
  preview: {
    host: '0.0.0.0',
    port: 4173,
    https: getHttpsConfig(),
  },
  // Define environment variables
  define: {
    'import.meta.env.VITE_HTTPS_ENABLED': JSON.stringify(!!getHttpsConfig()),
  },
});
