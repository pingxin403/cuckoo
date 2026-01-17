import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { HelloForm } from "./components/HelloForm";
import { TodoForm } from "./components/TodoForm";
import { TodoList } from "./components/TodoList";
import "./App.css";

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <div style={{ maxWidth: "1200px", margin: "0 auto", padding: "20px" }}>
        <h1 style={{ textAlign: "center", marginBottom: "40px" }}>
          Monorepo Hello/TODO Services
        </h1>

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "1fr 1fr",
            gap: "40px",
            marginBottom: "40px",
          }}
        >
          <div
            style={{
              border: "1px solid #ddd",
              borderRadius: "8px",
              padding: "20px",
            }}
          >
            <HelloForm />
          </div>

          <div
            style={{
              border: "1px solid #ddd",
              borderRadius: "8px",
              padding: "20px",
            }}
          >
            <TodoForm />
          </div>
        </div>

        <div
          style={{
            border: "1px solid #ddd",
            borderRadius: "8px",
            padding: "20px",
          }}
        >
          <TodoList />
        </div>
      </div>
    </QueryClientProvider>
  );
}

export default App;

