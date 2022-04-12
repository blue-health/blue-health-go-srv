create extension if not exists "uuid-ossp";

create table cards (
  id uuid primary key,
  state varchar (16) not null,
  data jsonb,
  inserted_at timestamp not null,
  updated_at timestamp not null
);
