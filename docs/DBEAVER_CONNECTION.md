# DBeaver Database Connection Configuration

## Mini-Kart PostgreSQL Database Connection

### Connection Details

Use the following configuration to connect to the Mini-Kart PostgreSQL database in DBeaver:

| Setting | Value |
|---------|-------|
| **Host** | `localhost` |
| **Port** | `5432` |
| **Database** | `minikart` |
| **Username** | `postgres` |
| **Password** | `postgres` |
| **SSL Mode** | `disable` |

### Step-by-Step Setup in DBeaver

1. **Open DBeaver** and click on the "New Database Connection" button (plug icon)

2. **Select PostgreSQL** from the database list and click "Next"

3. **Configure Connection Settings:**
   - **Host:** `localhost`
   - **Port:** `5432`
   - **Database:** `minikart`
   - **Authentication:** Database Native
   - **Username:** `postgres`
   - **Password:** `postgres`

4. **Configure Driver Properties (Optional):**
   - Click on "Driver Properties" tab
   - Ensure SSL mode is set to `disable`

5. **Test Connection:**
   - Click "Test Connection" button
   - If prompted to download driver files, click "Download"
   - You should see "Connected" message

6. **Save Connection:**
   - Click "Finish" to save the connection

### Connection String (for reference)

```
postgres://postgres:postgres@localhost:5432/minikart?sslmode=disable
```

### Database Schema Overview

Once connected, you should see the following tables:

- **products** - Product catalogue (13 sample products)
- **orders** - Customer orders
- **order_items** - Order line items

### Troubleshooting

#### Cannot Connect - Port Already in Use

If you cannot connect, ensure:

1. **Docker container is running:**
   ```bash
   docker ps | grep mini-kart-postgres
   ```

2. **No local PostgreSQL is running on port 5432:**
   ```bash
   lsof -i :5432
   ```

   If you see a local PostgreSQL process, stop it:
   ```bash
   brew services stop postgresql
   # or
   pkill postgres
   ```

3. **Restart Docker container if needed:**
   ```bash
   make docker-down
   make docker-up
   ```

#### Database "minikart" Does Not Exist

If DBeaver shows "database minikart does not exist", run migrations:

```bash
make migrate-up
```

#### Connection Timeout

Ensure the Docker container is healthy:

```bash
docker exec mini-kart-postgres pg_isready -U postgres
```

Expected output: `localhost:5432 - accepting connections`

### Verifying Database Content

After connecting, you can run these queries to verify the database:

```sql
-- Check all tables exist
SELECT tablename FROM pg_tables WHERE schemaname = 'public';

-- Count products
SELECT COUNT(*) FROM products;

-- View sample products
SELECT id, name, price, category FROM products LIMIT 5;

-- Check database extensions
SELECT * FROM pg_extension;
```

Expected results:
- 3 tables: products, orders, order_items
- 13 products in the products table
- uuid-ossp extension installed

### Security Note

⚠️ **Development Only**: These credentials are for local development only.
**Never use these default credentials in production!**
