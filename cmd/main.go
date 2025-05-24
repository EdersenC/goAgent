package main

import (
	"fmt"
	"github.com/EdersenC/goAgent/api"
	"os"
)

var agents = map[string]*api.Agent{}
var toolRegistry *api.ToolRegistry

func setUP() {
	agentsFolder, err := os.Open("agents.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	toolRegistry = api.NewToolRegistry()
	err = api.LoadAgents(agentsFolder, &agents)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func main() {
	setUP()
	plannerAgent := agents["Planner"]
	chat := api.NewChat(plannerAgent, toolRegistry)
	content := "What day was thursday?"
	chat.AddMessage("user", content)
	response, _ := plannerAgent.Chat(chat, false)
	println("Response:", response.StatusCode)

}
