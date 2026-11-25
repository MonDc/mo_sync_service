# Database Migrations

Database schema and version-controlled migrations for the MO service, managed via `golang-migrate`.

## 1. Create the Database Migration
Generate the initial pair of files for creating the database itself:
```bash
migrate create -ext sql -dir database/migrations -seq create_database
Write the CREATE DATABASE statement in the .up.sql file and DROP DATABASE in the .down.sql.

2. Create Table Migrations
Generate subsequent files for tables, columns, etc.:

bash
migrate create -ext sql -dir database/migrations -seq <migration_name>
3. Provision Database (One-time)
Because golang-migrate requires an existing DB to connect to, create it manually to keep tracking clean:

bash
mysql -h <host> -u <user> -p -e "CREATE DATABASE IF NOT EXISTS mo_db DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"
4. Apply / Rollback
bash
# Apply pending migrations
migrate -database 'mysql://user:pass@tcp(ip:3306)/mo_db' -path database/migrations up

# Rollback last migration
migrate -database 'mysql://user:pass@tcp(ip:3306)/mo_db' -path database/migrations down 1