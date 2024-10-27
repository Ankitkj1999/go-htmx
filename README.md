# Open PostgreSQL

```sh
psql -h <hostname> -U <username> -d <database>

CREATE USER quiz_user WITH PASSWORD 'your_password';

CREATE DATABASE quiz_db;

GRANT ALL PRIVILEGES ON DATABASE quiz_db TO quiz_user;

