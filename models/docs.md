# 🧭 **Regras Oficiais para Criação e Manutenção de Models (Go + Chi)**

**Padrões estruturais, de segurança, consistência e arquitetura**

Essas regras definem **como todos os models devem ser escritos**, como suas funções devem se comportar e como evoluções/manutenções devem ser realizadas sem quebrar o padrão da aplicação.

---

# 1. **Um Model é sempre uma struct simples que representa a linha de uma tabela**

Cada model deve:

✔ Representar uma linha da tabela de forma direta
✔ Ter campos que correspondam exatamente às colunas
✔ Ter tags JSON coerentes com o padrão snake_case do banco
✔ Não possuir métodos complexos

Exemplo padrão:

```go
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

**Nunca colocar lógica de negócio complexa dentro dos atributos.**

---

# 2. **Um model nunca conhece o router, controllers ou HTTP**

✔ Ele só deve trabalhar com:

* `context.Context`
* `*sql.Tx`
* queries SQL
* regras internas específicas à tabela

❌ Nunca retornar status HTTP
❌ Nunca ler *request*
❌ Nunca escrever *response*

> Models **não sabem que a web existe**.

---

# 3. **Todas as operações com banco devem sempre receber o contexto e a transação**

Obrigatório:

```go
func (u *User) Update(ctx context.Context, tx *sql.Tx) error
```

**Nunca:**

```go
func (u *User) Update() error          // errado
func (u *User) Update(ctx context.Context) error // errado
func (u *User) Update(tx *sql.Tx) error          // errado
```

O controller controla a transação.
O model apenas executa.

---

# 4. **Sempre usar ExecContext, QueryContext, QueryRowContext**

Obrigatório:

```go
_, err := ExecContext(tx, ctx, stmt, ...)
```

Esses wrappers já tratam timeout, logging e métricas da aplicação.

❌ Nunca usar `tx.Exec`, `tx.Query`, `tx.QueryRow`.

---

# 5. **Criação (`Create`) sempre deve gerar o ID dentro do model**

Padrão obrigatório:

```go
id := uuid.NewString()
```

* O model é o dono do ID
* O controller nunca gera IDs
* IDs devem usar sempre UUID v4 (`uuid.NewString()`)

---

# 6. **Nunca armazenar ou retornar senhas sem hash**

Regras obrigatórias:

✔ Hash sempre com:

```go
bcrypt.GenerateFromPassword([]byte(password), 12)
```

✔ Comparar com:

```go
bcrypt.CompareHashAndPassword(...)
```

✔ Nunca retornar o hash em qualquer método sem antes chamar `RemovePassword()`
✔ Nunca logar ou imprimir senhas

❌ Nunca armazenar senha em texto plano
❌ Nunca usar outro algoritmo de hash

---

# 7. **Métodos de UPDATE só podem atualizar as colunas diretamente relacionadas**

Exemplo:

```go
func (u *User) Update(ctx context.Context, tx *sql.Tx) error
```

Não pode:

* Criar outras entidades
* Alterar assinaturas (`invoices`, etc.)
* Enviar e-mail
* Fazer chamadas externas
* Aplicar lógica de negócio pesada

> O model é apenas um *gateway SQL*, não um serviço.

---

# 8. **Toda consulta (GetOne, GetAll, etc.) deve preencher o model completo**

Regras:

✔ Sempre usar `Scan(...)` com todos os campos necessários
✔ Sempre mapear TODOS os campos relevantes do model
✔ Campos calculados (booleans, formatados) devem ser feitos no SQL

Exemplo padrão (do código atual):

```sql
(cpfcnpj <> '' and address <> '') as profile_completed
```

Esse é o padrão correto:
**Regra de preenchimento de profile calculada no SQL, não no Go.**

---

# 9. **Nunca retornar listas de struct direto — sempre slices de ponteiros**

Padrão:

```go
[]*User
```

Regras:

✔ Evita cópias desnecessárias
✔ Ajuda no mapeamento e compatibilidade
✔ Facilita retornos em JSON

---

# 10. **Models nunca fazem validações estruturais (exceto hashing e cálculos simples)**

Permitido:

✔ Hash de senha
✔ Remover senha
✔ Validação interna de senha (`ValidatePassword`)
✔ Cálculos dentro do SQL (profile_completed, in_trial, …)

Proibido:

❌ Validar CPF
❌ Validar email duplicado
❌ Checar permissões (role)
❌ Checar limites de assinatura
❌ Regras de negócio complexas

Isso tudo pertence ao controller ou service.

---

# 11. **Métodos de DELETE devem respeitar o padrão da aplicação**

A aplicação não realiza DELETE físico.
O padrão é:

* “Anonimizar” dados sensíveis
* Nunca excluir linhas reais

Padrão obrigatório:

```go
update users set cpfcnpj = '', address = '' ...
```

Nunca usar:

```sql
DELETE FROM users
```

---

# 12. **Toda query deve ser multilinha, identada e com ordem lógica**

Exemplo correto:

```go
query := `
    select
        id,
        name,
        email
    from users
    where id = $1
`
```

Nunca colocar SQL em uma linha só.
Nunca interpolar strings manualmente.
Sempre usar placeholders `$1, $2, $3...`.

---

# 13. **Um model nunca inicia, commita ou faz rollback de transação**

Essa responsabilidade é sempre do controller:

❌ Proibido:

```go
tx.Commit()
tx.Rollback()
BeginTransaction(...)
```

✔ Sempre depender do que o controller enviar.

---

# 14. **Para cada tabela, o model deve conter obrigatoriamente:**

* `Create`
* `Update`
* `Delete`
* `GetOne`
* `GetAll`
* `RemoveSensitiveData / RemovePassword` (se houver dados sigilosos)
* Métodos auxiliares relevantes, mas sempre **focados na tabela em si**

Nada mais.

---

# 15. **Campos computados devem ser implementados no SQL, não no Go**

Exemplos válidos:

* profile_completed
* in_trial
* trial_end formatado

Evitar cálculos complexos fora do SQL, pois:

✔ O banco é otimizado para isso
✔ Reduz sobrecarga no Go
✔ Mantém padronização

---

# 16. **Limites de paginação: nunca colocar LIMIT/OFFSET fixos dentro do model**

O controller deve enviar:

```go
limit int
offset int
```

O model só aplica:

```go
limit $1 offset $2
```

Nunca decidir valores dentro do model.

---

# 17. **Todo model deve ser 100% unit testable**

Para isso:

✔ Não usar variáveis globais
✔ Não depender de estado externo
✔ Sempre receber `ctx` e `tx`
✔ Não fazer chamadas externas (HTTP, e-mail)

---

# ✔ **Checklist obrigatório do Model**

Antes de abrir PR:

* [ ] Struct mapeia corretamente a tabela
* [ ] Métodos seguem assinatura `(ctx context.Context, tx *sql.Tx)`
* [ ] Create gera UUID com `uuid.NewString()`
* [ ] Update e Delete não criam nem modificam outras entidades
* [ ] Nenhum SELECT/UPDATE/INSERT/DELETE fora de multiline SQL
* [ ] Queries usam placeholders
* [ ] Nunca há DELETE físico
* [ ] Sempre usa ExecContext / QueryContext / QueryRowContext
* [ ] Nenhum log, print ou acesso ao HTTP
* [ ] Campos computados feitos no SQL, não no Go
* [ ] Remoção de senha com `RemovePassword()` se necessário
* [ ] Código limpo, funções menores que ~80 linhas
* [ ] Nenhuma regra de negócio no model
