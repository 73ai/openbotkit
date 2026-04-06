import express from "express";
import { createProxyMiddleware } from "http-proxy-middleware";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const distDir = path.join(__dirname, "dist");

const app = express();
const port = parseInt(process.env.PORT || "3000", 10);
const apiTarget = process.env.API_URL || "http://localhost:8090";

app.use(
  "/api",
  createProxyMiddleware({
    target: apiTarget,
    changeOrigin: true,
  })
);

app.use(express.static(distDir));

app.get("/dashboard", (_req, res) => {
  res.sendFile(path.join(distDir, "dashboard.html"));
});

app.get("/usecase-form.html", (_req, res) => {
  res.sendFile(path.join(distDir, "usecase-form.html"));
});

app.get("/usecase.html", (_req, res) => {
  res.sendFile(path.join(distDir, "usecase.html"));
});

app.get("*", (_req, res) => {
  res.sendFile(path.join(distDir, "index.html"));
});

app.listen(port, () => {
  console.log(`Frontend server listening on :${port}, API proxy -> ${apiTarget}`);
});
