import React from "react";
import ReactDOM from "react-dom/client";
import { HashRouter } from "react-router-dom";
import App from "./App";
import { MDDMTestHarness } from "./test-harness/MDDMTestHarness";
import { initFeatureFlags } from "./features/featureFlags";
import "@fontsource/dm-sans/300.css";
import "@fontsource/dm-sans/400.css";
import "@fontsource/dm-sans/500.css";
import "@fontsource/dm-sans/600.css";
import "@fontsource/dm-mono/400.css";
import "@fontsource/dm-mono/500.css";
import "./styles.css";

// Fetch server-controlled feature flags before first render.
// .finally() ensures a network error still mounts the app (defaults apply).
// Dev-only: mount MDDMTestHarness before App so auth hooks never fire.
// Hash routing encodes the path in location.hash, e.g. /#/test-harness/mddm.
const isTestHarness =
  import.meta.env.DEV &&
  window.location.hash.startsWith("#/test-harness/mddm");

initFeatureFlags().finally(() => {
  ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
      {isTestHarness ? (
        <MDDMTestHarness />
      ) : (
        <HashRouter>
          <App />
        </HashRouter>
      )}
    </React.StrictMode>,
  );
});
