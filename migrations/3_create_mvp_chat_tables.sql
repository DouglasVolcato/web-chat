create table if not exists sessions (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    token_hash text not null,
    expires_at timestamptz not null,
    revoked_at timestamptz,
    created_at timestamptz not null default now()
);
create index if not exists sessions_user_id_idx on sessions(user_id);

create table if not exists contacts (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    contact_user_id uuid not null references users(id) on delete cascade,
    created_at timestamptz not null default now(),
    unique(user_id, contact_user_id),
    check (user_id <> contact_user_id)
);
create index if not exists contacts_user_id_idx on contacts(user_id);

create table if not exists chats (
    id uuid primary key default gen_random_uuid(),
    created_at timestamptz not null default now()
);

create table if not exists chat_participants (
    chat_id uuid not null references chats(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    created_at timestamptz not null default now(),
    primary key(chat_id, user_id)
);
create index if not exists chat_participants_user_idx on chat_participants(user_id);

create table if not exists messages (
    id uuid primary key default gen_random_uuid(),
    chat_id uuid not null references chats(id) on delete cascade,
    sender_user_id uuid not null references users(id) on delete cascade,
    content text not null,
    expires_at timestamptz not null,
    created_at timestamptz not null default now(),
    check (char_length(content) <= 2000)
);
create index if not exists messages_chat_created_idx on messages(chat_id, created_at desc);
create index if not exists messages_expires_idx on messages(expires_at);

create table if not exists qr_tokens (
    id uuid primary key default gen_random_uuid(),
    owner_user_id uuid not null references users(id) on delete cascade,
    token_hash text not null unique,
    expires_at timestamptz not null,
    used_by_user_id uuid references users(id) on delete set null,
    used_at timestamptz,
    created_at timestamptz not null default now()
);
create index if not exists qr_tokens_owner_idx on qr_tokens(owner_user_id);

create table if not exists push_subscriptions (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    endpoint text not null,
    p256dh text not null,
    auth text not null,
    consented_at timestamptz not null default now(),
    deleted_at timestamptz,
    created_at timestamptz not null default now(),
    unique(user_id, endpoint)
);

create table if not exists audit_logs (
    id uuid primary key default gen_random_uuid(),
    user_id uuid references users(id) on delete set null,
    action text not null,
    metadata jsonb not null default '{}'::jsonb,
    ip inet,
    created_at timestamptz not null default now()
);
create index if not exists audit_logs_action_idx on audit_logs(action, created_at desc);
