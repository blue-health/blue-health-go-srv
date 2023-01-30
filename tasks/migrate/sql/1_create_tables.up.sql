create extension if not exists "uuid-ossp";

create table cakes (
  id uuid primary key,
  identity_id uuid not null,
  name varchar (255) unique,
  inserted_at timestamp not null,
  updated_at timestamp not null
);

create index cakes_identity_id_index on cakes (identity_id);

create table cookies (
  id uuid primary key,
  identity_id uuid not null,
  inserted_at timestamp not null,
  updated_at timestamp not null
);

create index cookies_members_index on cookies (identity_id);
