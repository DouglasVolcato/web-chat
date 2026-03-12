package ai_tools

import (
	"app/ai/ai_agents"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/sashabaranov/go-openai"
)

type toolArgs struct {
	WeightKg float64 `json:"weight_kg"`
	HeightM  float64 `json:"height_m"`
}

type CalculateBMITool struct {
}

func (c *CalculateBMITool) run(args toolArgs) string {
	bmi := args.WeightKg / math.Pow(args.HeightM, 2)

	status := "Normal"
	if bmi < 18.5 {
		status = "Abaixo do peso"
	} else if bmi >= 25 && bmi < 30 {
		status = "Sobrepeso"
	} else if bmi >= 30 {
		status = "Obesidade"
	}

	return fmt.Sprintf("BMI: %.2f\nStatus: %s", bmi, status)
}

func (c *CalculateBMITool) GetDefinition() openai.FunctionDefinition {
	return openai.FunctionDefinition{
		Name:        "calculate_bmi",
		Description: "Calcula o índice de massa corporal (IMC) a partir de peso e altura",
		Parameters: json.RawMessage(`{
		"type": "object",
		"properties": {
			"weight_kg": {"type": "number", "description": "Peso em quilogramas"},
			"height_m": {"type": "number", "description": "Altura em metros"}
		},
		"required": ["weight_kg", "height_m"]
	}`),
	}
}

func (c *CalculateBMITool) Handle(functionCall string, arguments []byte) (ai_agents.ToolResult, error) {
	var args toolArgs
	if err := json.Unmarshal(arguments, &args); err != nil {
		log.Fatal("Erro parseando args:", err)
	}

	return ai_agents.ToolResult{Response: c.run(args), ReturnValueToAgent: false}, nil
}
