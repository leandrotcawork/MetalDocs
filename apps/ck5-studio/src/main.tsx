import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "ckeditor5/ckeditor5.css";
import "./styles/tokens.css";
import "./styles/base.css";
import "./styles/app.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
