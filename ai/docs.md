# 🧭 **Documentação Oficial — Sistema de Agentes e Ferramentas de IA (Go)**

O sistema de agentes de IA permite que a aplicação:

* Faça prompts mais estruturados
* Execute ferramentas internas via function calling
* Tenha respostas contextualizadas
* Delegue tarefas a ferramentas específicas
* Reencaminhe resultados ao modelo, quando necessário

O sistema é dividido em duas pastas:

```
/ai
  /ai_agents
      agent.go
  /ai_tools
      calculate_bmi_tool.go
```

---

# 1. **O que é um Agente?**

Um **Agente** é uma entidade que:

✔ Recebe um prompt do desenvolvedor
✔ Recebe mensagens do usuário
✔ Chama ferramentas quando o modelo solicitar
✔ Gera a resposta final para o app

## Estrutura do agente:

```go
type Agent struct {
    prompt string
    tools  []Tool
}
```

O agente possui:

* Um **prompt** base (sistema)
* Uma lista de **ferramentas registradas**

Ele é criado assim:

```go
agent := ai_agents.NewAgent("Você é um assistente...", []Tool{
    &CalculateBMITool{},
})
```

E é chamado assim:

```go
resp, err := agent.Answer(ctx, messages)
```

---

# 2. **Como funciona o fluxo interno de um Agente**

Quando o agente recebe mensagens:

1. O Agente junta:

   * prompt do sistema
   * mensagens anteriores
   * definições de funções das ferramentas

2. O agente envia tudo para o GPT 4o-mini com function calling habilitado.

3. O modelo pode responder de duas formas:

### ✔ A) Resposta direta (mensagem normal)

O agente retorna o texto imediatamente.

### ✔ B) Resposta pedindo execução de uma ferramenta (FunctionCall)

O agente então:

* Descobre qual ferramenta deve executar
* Roda o método `Handle()` da ferramenta
* A ferramenta retorna um `ToolResult`

O `ToolResult` define se a resposta:

* deve ser retornada diretamente ao usuário, ou
* deve ser enviada novamente ao LLM (return back to agent)

---

# 3. **Interface obrigatória para ferramentas (Tools)**

Todas as ferramentas devem implementar a interface:

```go
type Tool interface {
    GetDefinition() openai.FunctionDefinition
    Handle(functionCall string, arguments []byte) (ToolResult, error)
}
```

### ✔ `GetDefinition()`

Define:

* nome da ferramenta
* descrição
* schema JSON dos argumentos

### ✔ `Handle()`

Executa a ferramenta de fato.

---

# 4. **Criando uma nova ferramenta**

Toda ferramenta deve seguir este modelo:

### 4.1 Estrutura base

Criar um arquivo em:

```
/ai/ai_tools/<nome_da_tool>_tool.go
```

Ex.:

```
calculate_interest_tool.go
```

### 4.2 Exemplo de template oficial:

```go
package ai_tools

import (
	"app/ai/ai_agents"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type toolArgs struct {
	Value float64 `json:"value"`
	Rate  float64 `json:"rate"`
}

type CalculateInterestTool struct {}

func (t *CalculateInterestTool) run(args toolArgs) string {
	interest := args.Value * (args.Rate / 100)
	return fmt.Sprintf("Juros: %.2f", interest)
}

func (t *CalculateInterestTool) GetDefinition() openai.FunctionDefinition {
	return openai.FunctionDefinition{
		Name:        "calculate_interest",
		Description: "Calcula juros simples",
		Parameters: json.RawMessage(`{
		  "type": "object",
		  "properties": {
		    "value": {"type": "number"},
		    "rate":  {"type": "number"}
		  },
		  "required": ["value","rate"]
		}`),
	}
}

func (t *CalculateInterestTool) Handle(functionCall string, arguments []byte) (ai_agents.ToolResult, error) {
	var args toolArgs
	if err := json.Unmarshal(arguments, &args); err != nil {
		return ai_agents.ToolResult{}, err
	}

	return ai_agents.ToolResult{
		Response: t.run(args),
		ReturnValueToAgent: false,
	}, nil
}
```

### Regras obrigatórias:

* Arquivo deve ser snake_case + `_tool.go`
* Nome da struct deve ser PascalCase e terminar em `Tool`
* Os argumentos precisam ter struct específica
* JSON deve validar os argumentos
* O método `run()` implementa a lógica isolada
* `Handle()` nunca deve fazer log.Fatal
* `ReturnValueToAgent` define se o agente deve reenviar o resultado ao modelo

---

# 5. **Criando um novo agente**

Local:

```
/ai/ai_agents/agent.go
```

Criar um agente é simples:

```go
agent := ai_agents.NewAgent(
    "Você é um assistente especialista em saúde...",
    []ai_agents.Tool{
        &CalculateBMITool{},
        &CalculateInterestTool{},
    },
)
```

### Regras:

* Cada agente deve ter um prompt curto, claro e objetivo
* O agente não deve ter estado interno além do prompt e tools
* A lista de ferramentas deve ser pequena e específica para o agente

---

# 6. **Como chamar o agente no backend**

Exemplo:

```go
messages := []openai.ChatCompletionMessage{
	{Role: "user", Content: "Calcule o IMC de 80kg e 1.78m"},
}

resp, err := agent.Answer(ctx, messages)
if err != nil {
    return err
}

fmt.Println(resp)
```

---

# 7. **Como adicionar uma ferramenta nova a um agente existente**

Basta editar o arquivo onde o agente é criado:

```go
tools := []ai_agents.Tool{
    &CalculateBMITool{},
    &CalculateInterestTool{}, // nova
}
```

Não precisa alterar o agente em si.

---

# 8. **Regras obrigatórias de manutenção**

### ✔ Toda ferramenta deve ser:

* idempotente
* pura (sem side-effects inesperados)
* sem chamadas HTTP internas (isso deve ir para Services)
* segura para inputs estranhos

### ✔ Nunca usar `log.Fatal` dentro de ferramentas

Use `return ToolResult{}, err`.

### ✔ Não criar agentes genéricos “faz tudo”

Cada agente deve ser especialista em um domínio da aplicação.

---

# 9. **Como criar um agente novo**

Criar arquivo:

```
/ai/ai_agents/my_agent.go
```

Exemplo:

```go
package ai_agents

func NewMedicalAgent() Agent {
	return NewAgent(
		"Você é um agente especializado em diagnósticos médicos...",
		[]Tool{
			&CalculateBMITool{},
			&MedicalSymptomCheckerTool{},
		},
	)
}
```

Regras:

* Nome do agente = PascalCase e começar com `New...Agent()`
* Não colocar lógica além do construtor

---

# 10. **O ciclo completo do function calling**

### 1. Usuário envia mensagem

### 2. Agente envia ao modelo

### 3. Modelo decide:

* “responder” OU
* “chamar ferramenta X com essas informações”

### 4. Agente executa a ferramenta

### 5. Ferramenta devolve resultado

### 6. Agente decide:

* reenviar o resultado ao modelo (ReturnValueToAgent = true)
* retornar direto ao usuário (false)

### 7. Backend devolve resposta ao frontend

---

# 11. **Checklist de qualidade para novas ferramentas**

Antes de aprovar o PR, verificar:

* [ ] Nome da ferramenta segue `<feature>Tool`
* [ ] Arquivo segue `<feature>_tool.go`
* [ ] Estrutura toolArgs criada corretamente
* [ ] JSON dos parameters válido e completo
* [ ] Lógica isolada no método run()
* [ ] Handle() implementa parse → run → retorna ToolResult
* [ ] Não usa log.Fatal
* [ ] Não faz HTTP direto
* [ ] Não faz DB direto
* [ ] Documentação clara na descrição do function

---

# 12. **Checklist para novos agentes**

* [ ] Prompt simples, claro e objetivo
* [ ] Lista de tools pequena e específica
* [ ] Nome do agente claro
* [ ] Usar NewAgent()
* [ ] Não adicionar lógica extra no agente
* [ ] Agente não deve ter estado mutável
