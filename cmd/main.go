package main

import (
	"fmt"

	"github.com/EdersenC/gigaAi/api"
	"github.com/EdersenC/gigaAi/api/tools"
)




  
func main(){
    provider := api.NewProvider(
        "http://localhost:11434/api/", 
        "generate/",
        "chat", 
    )
   
    llama3:= &api.Agent{
        Name: "llama3.1:latest",
        Provider: provider,
    }
    agents := make([]*api.Agent,0)
    agents = append(agents,llama3 )

    registry := api.NewToolRegistry()

    tool:= api.NewToolParameters("function")
tool.AddProperty(
    "select_model",
    "string",
    `"Choose a model to use:
    'gemma': A fast and versatile general-purpose model balanced for speed and intelligence
    'contextguard': Specializes in contextual understanding, content filtering, and safety guardrails
    '* **Planner Agent Handoff:**
    * When the user's request requires generating a multi-step plan or strategy to achieve a goal.  For example, a user asking "How do I plan a trip to Europe?" might be handed off to a PlannerAgent.
    * When the current agent needs to decompose a complex task into smaller, manageable subtasks.
    * When the agent needs to create a sequence of actions involving multiple tools or other agents.

* **Searcher Agent Handoff:**
    * When the user's request requires retrieving information from external sources, such as the web, a database, or an API. For example, "What is the current weather in London?" would be handed off to a SearcherAgent.
    * When the current agent lacks the specific knowledge to answer a question.
    * When the agent needs to find relevant documents or data to support its reasoning or decision-making.

* **General Agent Handoff:**
    * When the user's request falls outside the specific expertise of the current agent, and a more general-purpose agent is needed.  This acts as a fallback.
    * When the current agent has completed a specialized task and the conversation needs to return to a more conversational or user-facing agent.
    * When the agent detects ambiguity in the user's request and needs to hand off to an agent that can clarify or gather more information.
    'prompt_curator': Specialized in refining, optimizing, and generating effective prompts"`,
    []string{"gemma", "contextguard", "searcher", "planner", "prompt_curator"},
    true,
)
    modelSelector := tools.NewHandOff(llama3, *tool)

    registry.RegisterTool(modelSelector)
    for i := 0; i < 1; i++ {

    agent := agents[i]
    agent.Tools = registry.GetTools()
    chat := api.NewChat(agent)
    message := &api.Message{
        Role: "user",
        Content: `"Create a marketing plan for a new coffee shop and what did donald tump do yesterdayy `,
    }

    chat.Messages = append(chat.Messages,message)

    response,_:= agent.Provider.Chat(agent,chat,false) 
    chatResponse,err:= api.DecodeChatResponse(response.Body)
    if err !=nil{
        fmt.Println(err)
    }
    fmt.Println(chatResponse)
    }



}









