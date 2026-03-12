create extension if not exists "pgcrypto";

create table if not exists users (
    id uuid primary key,
    name text not null,
    email text not null unique,
    password text not null default '',
    deleted_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists user_chats (
    id uuid primary key,
    user_id uuid not null references users(id) on delete cascade,
    title text not null,
    context text,
    deleted_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists user_chats_user_id_idx on user_chats(user_id);

create table if not exists user_chat_messages (
    id uuid primary key,
    chat_id uuid not null references user_chats(id) on delete cascade,
    role text not null,
    message text not null,
    emotion text not null default '',
    deleted_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists user_chat_messages_chat_id_idx on user_chat_messages(chat_id);
