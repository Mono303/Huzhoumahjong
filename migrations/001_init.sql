create table if not exists users (
  id text primary key,
  username varchar(32) not null,
  is_guest boolean not null default true,
  session_token text not null unique,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  last_seen_at timestamptz not null default now()
);

create table if not exists rooms (
  id text primary key,
  code varchar(12) not null unique,
  owner_user_id text not null,
  status varchar(16) not null,
  settings jsonb not null default '{}'::jsonb,
  match_id text null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists room_players (
  id text primary key,
  room_id text not null references rooms(id) on delete cascade,
  user_id text null references users(id) on delete set null,
  name varchar(32) not null,
  seat integer not null,
  is_host boolean not null default false,
  is_ready boolean not null default false,
  is_bot boolean not null default false,
  connected boolean not null default false,
  joined_at timestamptz not null default now(),
  unique (room_id, seat)
);

create index if not exists idx_room_players_room_id on room_players(room_id);
create index if not exists idx_room_players_user_id on room_players(user_id);

create table if not exists matches (
  id text primary key,
  room_id text not null references rooms(id) on delete cascade,
  status varchar(16) not null,
  winner_seat integer null,
  summary jsonb null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_matches_room_id on matches(room_id);
create index if not exists idx_matches_created_at on matches(created_at desc);

create table if not exists match_events (
  id text primary key,
  match_id text not null references matches(id) on delete cascade,
  sequence integer not null,
  event_type varchar(32) not null,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  unique (match_id, sequence)
);

create index if not exists idx_match_events_match_id on match_events(match_id);
