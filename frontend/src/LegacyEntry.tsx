import App from "./App";
import "./styles.css";

// Legacy global styles load with the legacy chunk so the redesign starts clean.
export default function LegacyEntry() {
  return <App />;
}
