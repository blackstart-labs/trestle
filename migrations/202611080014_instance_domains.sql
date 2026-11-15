ALTER TABLE users ADD COLUMN instance_operator boolean NOT NULL DEFAULT false;

CREATE TABLE instance_domain (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    public_origin text NOT NULL DEFAULT ''
);

INSERT INTO instance_domain DEFAULT VALUES;
