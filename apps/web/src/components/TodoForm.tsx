import { useState, type FormEvent } from "react";
import { useTodos } from "../hooks/useTodos";

export function TodoForm() {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const { createTodo, isCreating } = useTodos();

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();

    if (!title.trim()) {
      alert("Title is required");
      return;
    }

    createTodo(
      { title, description },
      {
        onSuccess: () => {
          // Clear form after successful creation
          setTitle("");
          setDescription("");
        },
        onError: (error) => {
          console.error("Failed to create todo:", error);
          alert("Failed to create todo. Please try again.");
        },
      }
    );
  };

  return (
    <div style={{ padding: "20px", maxWidth: "500px" }}>
      <h2>Create New TODO</h2>
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: "10px" }}>
          <label
            htmlFor="title"
            style={{ display: "block", marginBottom: "5px", fontWeight: "bold" }}
          >
            Title *
          </label>
          <input
            id="title"
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Enter todo title"
            required
            style={{
              width: "100%",
              padding: "8px",
              fontSize: "16px",
              border: "1px solid #ccc",
              borderRadius: "4px",
            }}
          />
        </div>

        <div style={{ marginBottom: "10px" }}>
          <label
            htmlFor="description"
            style={{ display: "block", marginBottom: "5px", fontWeight: "bold" }}
          >
            Description
          </label>
          <textarea
            id="description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Enter todo description (optional)"
            rows={4}
            style={{
              width: "100%",
              padding: "8px",
              fontSize: "14px",
              border: "1px solid #ccc",
              borderRadius: "4px",
            }}
          />
        </div>

        <button
          type="submit"
          disabled={isCreating}
          style={{
            padding: "10px 20px",
            fontSize: "16px",
            backgroundColor: isCreating ? "#ccc" : "#28a745",
            color: "white",
            border: "none",
            borderRadius: "4px",
            cursor: isCreating ? "not-allowed" : "pointer",
          }}
        >
          {isCreating ? "Creating..." : "Create TODO"}
        </button>
      </form>
    </div>
  );
}
