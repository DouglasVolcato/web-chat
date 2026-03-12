alter table if exists push_subscriptions
    add column if not exists status text not null default 'ACTIVE',
    add column if not exists user_agent_hash text,
    add column if not exists device_label text,
    add column if not exists last_seen_at timestamptz,
    add column if not exists revoked_at timestamptz;

update push_subscriptions
set status = case when deleted_at is not null then 'REVOKED' else 'ACTIVE' end
where status is null or status not in ('ACTIVE','INVALID','REVOKED');

alter table if exists push_subscriptions
    add constraint push_subscriptions_status_check check (status in ('ACTIVE','INVALID','REVOKED'));

create unique index if not exists push_subscriptions_user_endpoint_uidx on push_subscriptions(user_id, endpoint);
create index if not exists push_subscriptions_active_idx on push_subscriptions(user_id, status) where status = 'ACTIVE' and revoked_at is null;
