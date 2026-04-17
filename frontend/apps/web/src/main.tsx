import React from "react";
import ReactDOM from "react-dom/client";
import { HashRouter } from "react-router-dom";
import App from "./App";
import { MDDMTestHarness } from "./test-harness/MDDMTestHarness";
import { CK5TestHarness } from "./test-harness/CK5TestHarness";
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
const hash = window.location.hash;
const isMddmHarness = import.meta.env.DEV && hash.startsWith("#/test-harness/mddm");
const isCk5Harness = import.meta.env.DEV && hash.startsWith("#/test-harness/ck5");

initFeatureFlags().finally(() => {
  ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
      {isMddmHarness ? (
        <MDDMTestHarness />
      ) : isCk5Harness ? (
        <CK5TestHarness />
      ) : (
        <HashRouter>
          <App />
        </HashRouter>
      )}
    </React.StrictMode>,
  );
});
