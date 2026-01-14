set -gx DATABASE_URL "postgres://auth_user:password123@localhost:5432/auth_service?sslmode=disable"
set -gx JWT_SECRET "/2+XnmJGz1j3ehIVI/5P9kl+CghrE3DcS7rnT+qar5w"
set -gx PORT 8081
set -gx AMQP_URL "amqp://guest:guest@localhost:5672/"
set -gx LOGS_EXCHANGE "logs.events"
set -gx SERVICE_NAME "auth-service"
set -gx ENVIRONMENT "local"