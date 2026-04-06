import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@/styles/globals.css";
import UseCaseDetail from "@/pages/UseCaseDetail";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <UseCaseDetail />
  </StrictMode>
);
