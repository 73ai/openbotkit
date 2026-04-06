import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@/styles/globals.css";
import UseCaseForm from "@/pages/UseCaseForm";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <UseCaseForm />
  </StrictMode>
);
