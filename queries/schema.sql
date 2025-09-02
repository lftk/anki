CREATE TABLE col (
  id integer PRIMARY KEY,
  crt integer NOT NULL,
  mod integer NOT NULL,
  scm integer NOT NULL,
  ver integer NOT NULL,
  dty integer NOT NULL,
  usn integer NOT NULL,
  ls integer NOT NULL,
  conf text NOT NULL,
  models text NOT NULL,
  decks text NOT NULL,
  dconf text NOT NULL,
  tags text NOT NULL
);

CREATE TABLE notetypes (
  id integer NOT NULL PRIMARY KEY,
  name text NOT NULL,
  mtime_secs integer NOT NULL,
  usn integer NOT NULL,
  config blob NOT NULL
);

CREATE UNIQUE INDEX idx_notetypes_name ON notetypes (name);

CREATE INDEX idx_notetypes_usn ON notetypes (usn);

CREATE TABLE fields (
  ntid integer NOT NULL,
  ord integer NOT NULL,
  name text NOT NULL,
  config blob NOT NULL,
  PRIMARY KEY (ntid, ord)
) without rowid;

CREATE UNIQUE INDEX idx_fields_name_ntid ON fields (name, ntid);

CREATE TABLE templates (
  ntid integer NOT NULL,
  ord integer NOT NULL,
  name text NOT NULL,
  mtime_secs integer NOT NULL,
  usn integer NOT NULL,
  config blob NOT NULL,
  PRIMARY KEY (ntid, ord)
) without rowid;

CREATE UNIQUE INDEX idx_templates_name_ntid ON templates (name, ntid);

CREATE INDEX idx_templates_usn ON templates (usn);

CREATE TABLE notes (
  id integer PRIMARY KEY,
  guid text NOT NULL,
  mid integer NOT NULL,
  mod integer NOT NULL,
  usn integer NOT NULL,
  tags text NOT NULL,
  flds text NOT NULL,
  -- The use of type integer for sfld is deliberate, because it means that integer values in this
  -- field will sort numerically.
  sfld integer NOT NULL,
  csum integer NOT NULL,
  flags integer NOT NULL,
  data text NOT NULL
);

CREATE TABLE cards (
  id integer PRIMARY KEY,
  nid integer NOT NULL,
  did integer NOT NULL,
  ord integer NOT NULL,
  mod integer NOT NULL,
  usn integer NOT NULL,
  type integer NOT NULL,
  queue integer NOT NULL,
  due integer NOT NULL,
  ivl integer NOT NULL,
  factor integer NOT NULL,
  reps integer NOT NULL,
  lapses integer NOT NULL,
  left integer NOT NULL,
  odue integer NOT NULL,
  odid integer NOT NULL,
  flags integer NOT NULL,
  data text NOT NULL
);

CREATE TABLE revlog (
  id integer PRIMARY KEY,
  cid integer NOT NULL,
  usn integer NOT NULL,
  ease integer NOT NULL,
  ivl integer NOT NULL,
  lastIvl integer NOT NULL,
  factor integer NOT NULL,
  time integer NOT NULL,
  type integer NOT NULL
);

CREATE TABLE graves (
  usn integer NOT NULL,
  oid integer NOT NULL,
  type integer NOT NULL
);

CREATE TABLE config (
  KEY text NOT NULL PRIMARY KEY,
  usn integer NOT NULL,
  mtime_secs integer NOT NULL,
  val blob NOT NULL
) without rowid;

CREATE TABLE tags (
  tag text NOT NULL PRIMARY KEY COLLATE unicase,
  usn integer NOT NULL,
  collapsed boolean NOT NULL,
  config blob NULL
) without rowid;

CREATE TABLE decks (
  id integer PRIMARY KEY NOT NULL,
  name text NOT NULL COLLATE unicase,
  mtime_secs integer NOT NULL,
  usn integer NOT NULL,
  common blob NOT NULL,
  kind blob NOT NULL
);

CREATE UNIQUE INDEX idx_decks_name ON decks (name);

CREATE TABLE deck_config (
  id integer PRIMARY KEY NOT NULL,
  name text NOT NULL COLLATE unicase,
  mtime_secs integer NOT NULL,
  usn integer NOT NULL,
  config blob NOT NULL
);

-- syncing
CREATE INDEX ix_notes_usn ON notes (usn);

CREATE INDEX ix_cards_usn ON cards (usn);

CREATE INDEX ix_revlog_usn ON revlog (usn);

-- card spacing, etc
CREATE INDEX ix_cards_nid ON cards (nid);

-- scheduling and deck limiting
CREATE INDEX ix_cards_sched ON cards (did, queue, due);

-- revlog by card
CREATE INDEX ix_revlog_cid ON revlog (cid);

-- field uniqueness
CREATE INDEX ix_notes_csum ON notes (csum);

INSERT INTO
  col
VALUES
  (
    1,
    0,
    0,
    0,
    18,
    0,
    0,
    0,
    '',
    '',
    '',
    '',
    ''
  );