# 🧭 **Regras Oficiais para Criação de Controllers (Go + Chi + HTMX + EJS)**

**Com respostas sempre em HTML (views completas) ou fragmentos HTML para swap**

Essas regras garantem que **cada rota retorne conteúdo HTML**, permitindo que o frontend baseado em HTMX funcione de forma fluida, sem recarregar a página inteira.

---

# 1. **Todo controller deve seguir a estrutura base**

### ✔ Padrão de implementação:

```go
type SomethingController struct{}

func (c *SomethingController) RegisterRoutes(router chi.Router) {
    const path = "/something"

    router.Route(path, func(r chi.Router) {
        // middlewares
        // rotas
    })
}
```

### ✔ O controller apenas registra rotas

Nada de lógica fora dos handlers.

---

# 2. **Sempre usar rate limiting no escopo do controller**

```go
r.Use(httprate.LimitByIP(7, time.Minute))
```

---

# 3. **Rotas protegidas devem sempre usar `helpers.AuthDecorator`**

```go
r.Get("/x", helpers.AuthDecorator(func(w http.ResponseWriter, r *http.Request) {
    ...
}))
```

Nunca tratar o token manualmente.

---

# 4. **Sempre criar contexto com timeout no início do handler**

```go
ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
defer cancel()
```

Obrigatório em **todas** as rotas.

---

# 5. **Transações sempre com BeginTransaction → done()**

Padrão obrigatório:

```go
dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
if err != nil {
    return helpers.RenderErrorPage(w, err) // ou alert fragment
}
defer done()
```

Proibido:

❌ Commit
❌ Rollback manual
❌ Usar db direto

---

# 6. **Autenticação sempre via GetAuthUser**

```go
user, err := helpers.GetAuthUser(dbCtx, tx, r)
if err != nil {
    return helpers.RenderUnauthorized(w)
}
```

---

# 7. **Leitura de JSON só quando estritamente necessário**

A resposta do controller **não** deve ser JSON.
Mas o controller pode receber JSON se o envio for via HTMX, desde que:

✔ Use `ReadJSON`
✔ Retorne HTML na resposta

---

# 8. **Nunca retornar JSON como resposta final**

**Isso é proibido no frontend MPA HTMX.**

Todas as respostas devem ser uma destas:

### ✔ Página EJS completa

```go
helpers.Render(w, "app/decks/index.ejs", data)
```

### ✔ Fragmento para swap

```go
helpers.RenderFragment(w, "app/decks/cards-list.ejs", data)
```

### ✔ Fragmento contendo alerta de sucesso/erro

```go
helpers.RenderFragment(w, "partials/feedback.ejs", Alert{...})
```

### ❌ Proibido:

```go
helpers.WriteJSON(w, ...)
helpers.WriteErrorJSON(w, ...)
```

Esses métodos continuam existindo, mas são usados **apenas** pelas APIs internas
(`/api/...`) — **não** pelas views do app.

---

# 9. **Erros devem ser retornados como fragmento HTML**

### Nunca JSON

### Nunca texto puro

### Nunca http.Error

Opcões válidas:

### ✔ Alert com fragmento HTMX

```go
helpers.RenderFragment(w, "partials/feedback.ejs", feedback)
```

### ✔ Página de erro completa

```go
helpers.RenderErrorPage(w, err)
```

---

# 10. **Validações leves permitidas**

No controller é permitido validar:

* CPF/CNPJ
* Email duplicado
* Permissão por role
* Checagem de entidade pertence ao usuário

Mas a resposta deve ser:

✔ Um alerta HTML
ou
✔ Um fragmento substituindo parte da página

Exemplo:

```go
if !cpfValid {
    feedback := helpers.NewAlert("error", helpers.GetMessage("INVALID_CPF"))
    return helpers.RenderFragment(w, "partials/feedback.ejs", feedback)
}
```

---

# 11. **Chamadas externas devem usar outro contexto**

Mesmo padrão, mas resposta continua sendo HTML.

---

# 12. **Exclusões devem retornar fragmentos HTML para swap**

Padrão:

```go
helpers.RenderFragment(w, "app/users/user-row.ejs", updatedList)
```

Nunca JSON.

---

# 13. **Paginação deve retornar apenas o fragmento da lista**

Rota:

```
GET /resource/get-all
```

Controller deve retornar:

```go
helpers.RenderFragment(w, "app/resource/list.ejs", data)
```

HTMX fará o swap no container correto.

---

# 14. **Nunca colocar regra de negócio no controller**

Mesmo padrão anterior.

Mas agora:

O controller deve apenas:

* buscar dados
* atualizar banco
* montar estrutura
* renderizar o fragmento EJS correto

---

# 15. **Nomenclatura de endpoints segue padrão, mas respostas são HTML**

| Ação            | Rota                                  | Resposta       |
| --------------- | ------------------------------------- | -------------- |
| Criar           | `POST /resource/create`               | fragmento HTML |
| Atualizar       | `PATCH /resource/update`              | fragmento HTML |
| Excluir         | `DELETE /resource/delete`             | fragmento HTML |
| Excluir próprio | `DELETE /resource/delete-my-resource` | fragmento HTML |
| Listar          | `GET /resource/get-all`               | fragmento HTML |
| Ver             | `GET /resource/get-by-id`             | página EJS     |

---

# 16. **Cada rota é função anônima**

Idem regras anteriores.

---

# 17. **Nenhum handler pode ultrapassar 120 linhas**

Se ultrapassar:

➡ Mover processamento para service
➡ Manter controller fino

---

# 18. **Controladores do app nunca fazem redirecionamento com status 301/302**

HTMX deve decidir quando recarregar página ou trocar fragmentos.

Permitido:

```go
hx-push-url="true"
```

Proibido:

```go
http.Redirect(...)
```

---

# 19. **Sempre retornar conteúdo pronto para HTMX fazer o swap**

Essa é a regra mais importante.

Exemplos válidos:

### ✔ Retornar lista atualizada

```go
helpers.RenderFragment(w, "app/decks/cards-list.ejs", data)
```

### ✔ Retornar card recém-criado

```go
helpers.RenderFragment(w, "app/decks/card.ejs", newCard)
```

### ✔ Retornar alerta

```go
helpers.RenderFragment(w, "partials/feedback.ejs", feedback)
```

---

# ✔ Checklist Final dos Controllers (versão HTMX + Views)

Antes de aprovar PR, verificar:

* [ ] Resposta SEMPRE é HTML (view ou fragmento)
* [ ] Nunca retorna JSON
* [ ] Usa Render / RenderFragment
* [ ] Mantém padrão de partials
* [ ] Usa alertas HTML para erros
* [ ] Usa alertas HTML para sucesso
* [ ] Usa HTMX para swaps e updates
* [ ] Não contém redirect
* [ ] Controller fino, sem regra de negócio
* [ ] Transações corretas
* [ ] Timeout correto
* [ ] 120 linhas máximo por handler
* [ ] Usa AuthDecorator quando necessário
