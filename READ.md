# 🧠 goAgent (WIP)

> A modular Go SDK for building intelligent, LLM-powered agents and tools — fast.

## 📌 Overview

**goAgent** is a work-in-progress SDK for creating applications that interact with large language models (LLMs) using Go. It’s built to give developers the flexibility to prototype, extend, and productionize LLM agents — all while staying in the Go ecosystem.

With goAgent, you can:

- Spin up multi-tool chat agents in seconds
- Seamlessly switch between local (e.g. Ollama) and remote (e.g. OpenAI, Gemini) models
- Build reusable **tools** and **agents** using JSON or Go — with full interconversion
- Extend apps with dynamic functionality like real-time search, embedding, summarization, and more

---

## ✨ Features

- 🔌 Multi-backend LLM support (Ollama now; OpenAI, Gemini,...)
- 🔧 Pluggable **tooling system** — build tools in code or JSON
- 📄 Fully JSON-driven **agent configurations** (with Go ↔ JSON syncing)
- 🔍 Built-in search tool using **DuckDuckGo**(swappable) + embeddings-based relevance
- 🧠 Coming soon: MCP, memory chaining, and goal decomposition

---

## ⚙️ Agents & Tools: Code ↔ JSON

One of goAgent's core principles is **interoperability** between static code and dynamic configs.

✅ You can:

- Define agents and tools **in Go**
- Export/save them to JSON
- Load from JSON at runtime (for editing, sharing, hot-swapping)
- Combine both approaches in the same app

### 🧠 Example: Loading Agents

You can define all your agents in an `agents.json` file, and load them like this:

```go
toolRegistry := goAgent.NewToolRegistry()
goAgent.LoadAgentsFromJSON("agents.json", &agents)

goAgent.PlannerAgent = agents["Planner"]
goAgent.EmbeddingAgent = agents["Embedder"]
goAgent.SummaryAgent = agents["Summarizer"]
```

The result? A flexible, declarative agent system that’s perfect for modular apps or CLI interfaces.

---

## 🔧 Tooling System

Tools give agents the ability to *do things* — call APIs, fetch data, run calculations, or interact with users and files.

They are defined as **function-like-style**(Ollama api tooling) with:

- A name and description
- Input parameters (structured)
- Usage examples and constraints (to guide model behavior)

You can register tools:

- Programmatically in Go (`RegisterTools`)
- From external `.json` files (preferred for flexibility)

---

### 🔍 Example: Search Tool (JSON Schema)(WIP)

```json
{
  "type": "function",
  "function": {
    "name": "search",
    "description": "Access real-time web information using DuckDuckGo. Use this when the user asks about events, facts, or updates that may have occurred after your general knowledge cutoff — or when fresh, external data is clearly needed.",
    "examples": [
      "User: What are the impacts of climate change on agriculture?\nQueries:\n- 'climate change effects on crop yield'\n- 'drought impact on farming'"
    ],
    "constraints": [
      "Only generate as many queries as necessary — avoid filler or duplication.",
      "Use one query when the user is asking a specific, factual question.",
      "Avoid vague language. Be specific and context-aware.",
      "**Always prefer using this function over guessing when your internal knowledge may be outdated.**"
    ],
    "parameters": {
      "type": "object",
      "properties": {
        "queries": {
          "type": "array",
          "items": { "type": "string" },
          "description": "1 to 10 well-phrased search queries"
        },
        "reason": {
          "type": "string",
          "description": "Explain why these queries were chosen and how they relate to the user’s question."
        }
      },
      "required": ["queries", "reason"]
    }
  }
}
```

This tool allows your agent to call real-time search intelligently, especially for time-sensitive or external questions.

---

## 🧪 Example Usage

Once your agents and tools are initialized:

```go
chat := goAgent.NewChat(goAgent.PlannerAgent, toolRegistry)
response := chat.SendUserMessage("What's the latest with AI regulation?")
response.PrintContent()
```

Behind the scenes, the agent might:

- Generate queries using the search tool
- Embed and rank content by relevance
- Summarize and return a focused response

---

## 🛣️ Roadmap

- [x] Ollama integration (local model inference)
- [ ] DuckDuckGo search tool w/ embedding relevance(50%)
- [x] JSON agent/tool system
- [ ] OpenAI / Gemini support
- [ ] Wikipedia, YouTube, Google Search tools
- [ ] MCP system (multi-agent context routing)
- [ ] File tools (RAG, notes, memory recall)

---

## 🤝 Contributing

This SDK is actively being built to learn and build LLM driven apliactions — if you:

- Use Go
- Are curious about LLMs
- Want to prototype tools or agent systems

Feel free to open issues, ideas, or PRs! def need ideas on architecture

---

## 📜 License

MIT License
