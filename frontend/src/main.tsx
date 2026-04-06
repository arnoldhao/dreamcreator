import ReactDOM from "react-dom/client";

import { AppProviders } from "./app/providers/AppProviders";
import "./index.css";
import App from "./App";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <AppProviders>
    <App />
  </AppProviders>
);
