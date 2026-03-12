# 🧭 **Guia Oficial — Padrões de Views (EJS + HTMX + Bootstrap)**

As regras abaixo devem ser seguidas **em toda criação e manutenção** de páginas EJS dentro da pasta `presentation/views`.

Elas garantem:

* UI consistente
* UX fluida com HTMX
* Zero page reload desnecessário
* Componentização
* Código fácil de manter
* Layout padrão entre todas as páginas

---

# 1. **Toda view deve começar com `head.ejs` e terminar com `scripts.ejs`**

Padrão obrigatório:

```ejs
<%- include('../partials/head', { title }) %>
<body>
    ...
    <%- include('../partials/scripts') %>
</body>
```

* Nenhum arquivo deve incluir `<html>`, `<!DOCTYPE>` ou `<head>` diretamente.
* Isso vem sempre do partial.

---

# 2. **Uso obrigatório do navbar e footer nos layouts do app**

Para páginas internas (MPA):

```ejs
<%- include('../partials/navbar', { active, user, hideNavbar }) %>
...
<%- include('../partials/footer') %>
```

Para landing pages:

```ejs
<%- include('../partials/head-landing', { title }) %>
<%- include('../partials/footer') %>
```

**Nunca duplicar código de header/footer dentro de cada página.**

---

# 3. **HTML sempre organizado com container principal dentro de `<main>`**

Padrão:

```ejs
<main class="flex-grow-1">
    <div class="container d-flex flex-column gap-4">
        ...
    </div>
</main>
```

* Todas as páginas internas devem usar `.container` + `gap-*`
* Layout mínimo com flex column e altura total

---

# 4. **Uso obrigatório do sistema de partials para páginas modulares**

Componentes repetidos devem ficar em `/partials`:

* `navbar.ejs`
* `feedback.ejs`
* `scripts.ejs`
* `cards-list.ejs` (como no exemplo)
* breadcrumbs
* modais reutilizáveis
* formulários compartilhados

**Regra de ouro:**
➡️ *Se você repetiu o mesmo trecho de HTML 2 vezes → mova para partial.*

---

# 5. **HTMX é obrigatório para interações. Reload é proibido.**

### ❌ Nunca usar:

* `location.reload()`
* `window.location = ...`
* `window.href = ...`

### ✔ Sempre usar HTMX para:

* Atualizar partes da página (`hx-target`, `hx-swap`)
* Criar, deletar e editar itens
* Filtrar conteúdo
* Formular interactions com backend

Exemplo correto:

```ejs
<form 
  hx-post="/app/decks/<%= deck.id %>/cards"
  hx-target="#cardsList"
  hx-swap="beforebegin"
>
```

---

# 6. **Padrão obrigatório de swaps**

Use corretamente:

| Comportamento                    | HTMX                   |
| -------------------------------- | ---------------------- |
| Substituir conteúdo              | `hx-swap="innerHTML"`  |
| Inserir antes de uma lista       | `beforebegin`          |
| Inserir depois                   | `afterend`             |
| Recarregar somente um componente | `outerHTML`            |
| Resetar formulário após submit   | `hx-on::after-request` |

**Nunca substituir o HTML inteiro da página.**

---

# 7. **Nunca esconder elementos com JavaScript manual se Bootstrap resolve**

❌ Proibido:

```js
document.getElementById('modal').style.display = "none"
```

✔ Correto:

```ejs
<button class="btn-close" data-bs-dismiss="modal"></button>
```

Ou via HTMX:

```ejs
hx-on::after-request="bootstrap.Modal.getInstance(document.getElementById('createCardModal')).hide()"
```

---

# 8. **Feedback e alerts sempre pelo partial `feedback.ejs`**

### Nunca criar alertas inline com render estático.

Utilizar a infraestrutura existente:

```ejs
<%- include('../partials/feedback') %>
```

E no backend: `helpers.WriteJSON(w, ...)` retorna mensagens que são consumidas por `feedback.js`.

### Nunca criar novos sistemas de toast/alerts duplicados.

---

# 9. **Formulários sempre com semântica correta + Bootstrap**

Regras:

✔ Sempre declarar `<label>` corretamente
✔ Nunca deixar campos sem `id`
✔ Sempre usar classes do Bootstrap (`form-control`, `form-select`)
✔ Campos sempre agrupados em `row` + `col-*`

Exemplo:

```ejs
<div class="col-md-6">
  <label class="form-label fw-semibold" for="difficulty">Dificuldade inicial</label>
  <select class="form-select" id="difficulty" name="difficulty" required></select>
</div>
```

---

# 10. **Botões de ação devem usar o sistema de Loading Buttons**

Regras:

✔ Adicionar o atributo `data-loading-button`
✔ Manter marcações internas:

```ejs
<button class="btn btn-primary" data-loading-button>
  <span data-loading-label>Salvar</span>
  <span class="d-none" data-loading-spinner>
    <span class="spinner-border spinner-border-sm"></span>
    <span class="ms-2">Carregando...</span>
  </span>
</button>
```

Isso garante loading consistente na aplicação inteira.

---

# 11. **Modais devem sempre seguir este padrão:**

```ejs
<div class="modal fade" id="modalId" tabindex="-1" aria-hidden="true">
  <div class="modal-dialog modal-dialog-centered modal-lg">
    <div class="modal-content border-0 shadow">
        ...
    </div>
  </div>
</div>
```

Regras:

* Sempre usar Bootstrap Modal
* Sempre ter `.btn-close`
* Sempre fechar modal via Bootstrap, nunca hide manual
* Eventos via HTMX para fechar após sucesso

---

# 12. **Sempre usar estruturas acessíveis**

Obrigatório:

* `aria-label`, `aria-current`, `aria-controls`
* `role="status"` para spinners
* `aria-valuenow/aria-valuemin/aria-valuemax` em progress bars
* breadcrumbs com `<nav aria-label="breadcrumb">`

Isso garante acessibilidade mínima.

---

# 13. **Estrutura de pastas deve ser respeitada sempre**

```
presentation/public
presentation/views/app
presentation/views/landing
presentation/views/partials
```

### ✔ Views internas → `views/app`

### ✔ Landing pages → `views/landing`

### ✔ Itens reutilizáveis → `views/partials`

Nunca misturar.

---

# 14. **CSS personalizado sempre em `styles.css`**

Regras:

❌ Nunca colocar `<style>` dentro dos EJS.
❌ Nunca aplicar estilo inline.

✔ Todo estilo deve ir para `presentation/public/css/styles.css`.

---

# 15. **JavaScript personalizado deve ir em `presentation/public/js/app.js`**

Regras:

* Nada de JS inline nas views (exceto htmx events simples)
* Nada de script embaixo das views
* Toda lógica deve ser centralizada

Somente eventos simples (uma linha) podem aparecer no `hx-on::something`.

---

# 16. **Usar partial `scripts.ejs` para carregar JS**

Nunca carregar scripts manualmente.

---

# 17. **Sempre usar classes utilitárias do Bootstrap antes de inventar novas**

Ordem de prioridade:

1. Bootstrap spacing/layout (`d-flex`, `gap-3`, `container`, `row`, etc.)
2. Estilos globais em `styles.css`
3. Classes específicas minimalistas

Nunca inventar classes quando o Bootstrap já resolve.

---

# 18. **Não fixar cores de texto para suportar temas claro/escuro**

Regras:

* Evite aplicar classes que forcem texto sempre em preto/branco.
* Prefira classes sem cor fixa (`text-secondary`, `text-muted`) ou use as utilitárias do Bootstrap que respeitam o tema.
* Deixe a escolha de cores para o tema ativo.

---

# ✔ Checklist para PR de views

Antes de validar um PR:

* [ ] Página inicia com `head.ejs`
* [ ] Página termina com `scripts.ejs`
* [ ] navbar e footer são usados corretamente
* [ ] HTML está dentro de `<main>` + `.container`
* [ ] Não há `location.reload()`
* [ ] Todos os forms usam HTMX
* [ ] Targets e swaps corretos
* [ ] Modais seguem padrão Bootstrap
* [ ] Nada de CSS inline
* [ ] Nada de JS inline (salvo eventos htmx)
* [ ] Feedback usando `feedback.ejs`
* [ ] Código bem identado e clean
* [ ] Nenhuma duplicação estrutural (usar partials)
* [ ] Acessibilidade básica presente
