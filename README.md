# PS Club Backend

This is the backend for the PS Club CRM application.

## Environment Variables

The application can be configured using the following environment variables:

### Database Configuration
- `DB_HOST`: The hostname of the database server. (Default: `localhost`)
- `DB_PORT`: The port number of the database server. (Default: `5432`)
- `DB_USER`: The username for connecting to the database. (Default: `ps_club_user`)
- `DB_PASSWORD`: The password for connecting to the database. (Default: `ps_club_password`)
- `DB_NAME`: The name of the database. (Default: `ps_club_crm_db`)
- `DB_SSLMODE`: The SSL mode for connecting to the database. (Default: `disable`)
- `DB_SCHEMA_PATH`: The path to the database schema file. (Default: `""`)

### Server Configuration
- `PORT`: The port number for the server to listen on. (Default: `8080`)

### CORS Configuration
- `CORS_ALLOWED_ORIGINS`: A comma-separated list of allowed origins for CORS. (Default: `http://localhost:3000,http://localhost:3001`)
