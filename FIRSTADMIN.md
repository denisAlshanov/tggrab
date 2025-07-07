# Creating the First Super Admin User

This guide explains how to create the first super admin user in the St. Planer system after initial deployment.

## Prerequisites

1. **Service Running**: Ensure the St. Planer service is running and accessible
2. **Database Initialized**: Confirm that database migrations have completed successfully
3. **API Key**: You'll need a valid API key configured in your environment
4. **curl**: Ensure curl is installed on your system

## Step 1: Verify Service Health

First, verify that the service is running and the database is properly initialized:

```bash
curl -X GET http://localhost:8080/health
```

Expected response should show all services as healthy and migrations completed:
```json
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "storage": "healthy"
  },
  "migrations": {
    "current_version": 10,
    "migrations_applied": [...]
  }
}
```

## Step 2: Get the Super Admin Role ID

The system creates default roles during migration. First, retrieve the super_admin role ID:

```bash
curl -X POST http://localhost:8080/api/v1/roles/list \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "filters": {
      "search": "super_admin"
    },
    "pagination": {
      "page": 1,
      "limit": 1
    }
  }'
```

Expected response:
```json
{
  "success": true,
  "data": {
    "roles": [
      {
        "id": "ROLE_UUID_HERE",
        "name": "super_admin",
        "description": "Full system access",
        "permissions": [
          "users:create", "users:read", "users:update", "users:delete",
          "roles:create", "roles:read", "roles:update", "roles:delete",
          ...
        ],
        "status": "active",
        "user_count": 0,
        "created_at": "2025-01-07T00:00:00Z"
      }
    ],
    "total": 1,
    "pagination": {
      "page": 1,
      "limit": 1,
      "total": 1,
      "total_pages": 1
    }
  }
}
```

Save the `id` value from the response - you'll need it in the next step.

## Step 3: Create the First Super Admin User

Now create your first super admin user with the role ID from the previous step:

```bash
curl -X POST http://localhost:8080/api/v1/users/add \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "name": "Admin",
    "surname": "User",
    "email": "admin@gmail.com",
    "password": "YourSecurePassword123!",
    "role_ids": ["ROLE_UUID_HERE"],
    "metadata": {
      "created_by": "initial_setup",
      "is_first_admin": true
    }
  }'
```

### Important Password Requirements:
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one digit
- At least one special character

### Response:
```json
{
  "success": true,
  "data": {
    "id": "USER_UUID_HERE",
    "name": "Admin",
    "surname": "User",
    "email": "admin@gmail.com",
    "status": "active",
    "roles": [
      {
        "id": "ROLE_UUID_HERE",
        "name": "super_admin",
        "description": "Full system access"
      }
    ],
    "created_at": "2025-01-07T00:00:00Z",
    "updated_at": "2025-01-07T00:00:00Z"
  }
}
```

## Step 4: Verify the Super Admin User

Verify that the user was created successfully by retrieving their information:

```bash
curl -X GET http://localhost:8080/api/v1/users/info/USER_UUID_HERE \
  -H "X-API-Key: YOUR_API_KEY"
```

## Alternative: Using Environment Variables

For production deployments, you can create a script that uses environment variables:

```bash
#!/bin/bash

# Set environment variables
API_URL="${API_URL:-http://localhost:8080}"
API_KEY="${API_KEY}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@gmail.com}"
ADMIN_PASSWORD="${ADMIN_PASSWORD}"
ADMIN_NAME="${ADMIN_NAME:-Admin}"
ADMIN_SURNAME="${ADMIN_SURNAME:-User}"

# Check required variables
if [ -z "$API_KEY" ]; then
    echo "Error: API_KEY environment variable is required"
    exit 1
fi

if [ -z "$ADMIN_PASSWORD" ]; then
    echo "Error: ADMIN_PASSWORD environment variable is required"
    exit 1
fi

# Step 1: Get super_admin role ID
echo "Fetching super_admin role ID..."
ROLE_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/roles/list" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "filters": {"search": "super_admin"},
    "pagination": {"page": 1, "limit": 1}
  }')

ROLE_ID=$(echo "$ROLE_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$ROLE_ID" ]; then
    echo "Error: Could not find super_admin role"
    echo "Response: $ROLE_RESPONSE"
    exit 1
fi

echo "Found super_admin role ID: $ROLE_ID"

# Step 2: Create super admin user
echo "Creating super admin user..."
USER_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/users/add" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d "{
    \"name\": \"$ADMIN_NAME\",
    \"surname\": \"$ADMIN_SURNAME\",
    \"email\": \"$ADMIN_EMAIL\",
    \"password\": \"$ADMIN_PASSWORD\",
    \"role_ids\": [\"$ROLE_ID\"],
    \"metadata\": {
      \"created_by\": \"initial_setup\",
      \"is_first_admin\": true
    }
  }")

# Check if user was created successfully
if echo "$USER_RESPONSE" | grep -q '"success":true'; then
    echo "✅ Super admin user created successfully!"
    echo "Email: $ADMIN_EMAIL"
    USER_ID=$(echo "$USER_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo "User ID: $USER_ID"
else
    echo "❌ Failed to create super admin user"
    echo "Response: $USER_RESPONSE"
    exit 1
fi
```

Save this script as `create-first-admin.sh` and run it:

```bash
chmod +x create-first-admin.sh

# Run with environment variables
API_KEY="your-api-key" \
ADMIN_EMAIL="youradmin@gmail.com" \
ADMIN_PASSWORD="YourSecurePassword123!" \
./create-first-admin.sh
```

## Security Considerations

1. **Strong Password**: Always use a strong, unique password for the super admin account
2. **Secure Storage**: Store the admin credentials securely (use a password manager)
3. **HTTPS**: In production, always use HTTPS instead of HTTP
4. **API Key**: Keep your API key secure and rotate it regularly
5. **Audit**: The metadata field helps track that this was the initial admin setup

## Troubleshooting

### Error: "Email must be a Gmail account"
The system requires Gmail accounts. Ensure your email ends with `@gmail.com`.

### Error: "Invalid password"
Check that your password meets all requirements:
- At least 8 characters
- Contains uppercase, lowercase, digit, and special character

### Error: "Role not found"
Ensure database migrations completed successfully. Check the health endpoint to verify migration status.

### Error: "User already exists"
If a user with that email already exists, you'll need to use a different email or update the existing user.

## Next Steps

After creating the first super admin user:

1. **Login**: Implement authentication to get JWT tokens (if applicable)
2. **Create Additional Admins**: Use the super admin account to create other administrative users
3. **Create Standard Roles**: Set up additional roles for different user types
4. **Configure Permissions**: Fine-tune permissions for each role as needed

## Example: Creating Additional Admin Users

Once you have the first super admin, you can create additional admin users:

```bash
# Get the 'admin' role ID (not super_admin)
curl -X POST http://localhost:8080/api/v1/roles/list \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "filters": {"search": "admin"},
    "pagination": {"page": 1, "limit": 10}
  }'

# Create a regular admin user
curl -X POST http://localhost:8080/api/v1/users/add \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_API_KEY" \
  -d '{
    "name": "John",
    "surname": "Doe",
    "email": "john.doe@gmail.com",
    "password": "SecurePassword456!",
    "role_ids": ["ADMIN_ROLE_UUID"],
    "metadata": {
      "created_by": "super_admin"
    }
  }'
```

## API Reference

For complete API documentation, refer to the Swagger UI at:
```
http://localhost:8080/swagger/index.html
```

The relevant endpoints for user management are:
- `POST /api/v1/users/add` - Create new user
- `GET /api/v1/users/info/{user_id}` - Get user information
- `PUT /api/v1/users/update` - Update user information
- `POST /api/v1/users/list` - List all users
- `POST /api/v1/roles/list` - List all roles