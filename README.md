# UAS Achievement System API

Sistem pelaporan prestasi mahasiswa dengan Role-Based Access Control (RBAC) menggunakan Go, Fiber, dan MongoDB.

## Fitur

- **Manajemen Pengguna dengan RBAC**
  - Admin: Full access ke semua fitur
  - Mahasiswa: Create, read, update prestasi sendiri
  - Dosen Wali: Read, verify prestasi mahasiswa bimbingannya

- **Pelaporan Prestasi dengan Field Dinamis**
  - Kompetisi (competition)
  - Publikasi (publication)
  - Organisasi (organization)
  - Sertifikasi (certification)

- **Verifikasi Prestasi oleh Dosen Wali**
- **Dashboard dan Statistik Prestasi**

## Tech Stack

- **Backend**: Go 1.24+ dengan Fiber Framework
- **Database**: Hybrid Architecture
  - **PostgreSQL**: Master data (users, roles, students, lecturers)
  - **MongoDB**: Achievement data (achievements, references)
- **Authentication**: JWT
- **Password Hashing**: bcrypt

## Setup

1. **Clone repository**
```bash
git clone <repository-url>
cd UASBE
```

2. **Install dependencies**
```bash
go mod tidy
```

3. **Setup Database (Hybrid Architecture)**

   **PostgreSQL (Required - Master Data)**
   - Install PostgreSQL
   - Create database: `createdb uasbe`
   - Used for: users, roles, students, lecturers

   **MongoDB (Required - Achievement Data)**
   - Install MongoDB
   - Start MongoDB service
   - Used for: achievements, achievement references

4. **Environment Configuration**
   Copy `.env` file dan sesuaikan konfigurasi untuk hybrid database:

```env
# PostgreSQL Configuration (Master Data)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=uasbe

# MongoDB Configuration (Achievement Data)
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE_NAME=uasbe_achievements

# Application Configuration
APP_PORT=3000
JWT_SECRET=your-secret-key-change-in-production-make-it-very-long-and-secure
```

5. **Run Migrations**
   
   Jalankan migration untuk kedua database:

   **PostgreSQL (Master Data):**
```bash
# Linux/Mac  
./scripts/migrate.sh postgres

# Windows
scripts\migrate.bat postgres

# Or directly
go run cmd/migrate/main.go -db=postgres
```

   **MongoDB (Achievement Data):**
```bash
# Linux/Mac
./scripts/migrate.sh mongo

# Windows
scripts\migrate.bat mongo

# Or directly
go run cmd/migrate/main.go -db=mongo
```

6. **Run Application**
```bash
go run main.go
```

Server akan berjalan di `http://localhost:3000`

## API Endpoints

### Authentication

#### POST /api/auth/register
Register user baru
```json
{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "password123",
  "full_name": "John Doe",
  "role": "student",
  "student_id": "123456789",
  "program_study": "Teknik Informatika",
  "academic_year": "2024"
}
```

#### POST /api/auth/login
Login user
```json
{
  "username": "john_doe",
  "password": "password123"
}
```

### Achievements

#### POST /api/achievements
Buat prestasi baru (requires authentication)
```json
{
  "category": "competition",
  "title": "Juara 1 Programming Contest",
  "description": "Memenangkan kompetisi programming tingkat nasional",
  "details": {
    "competition_name": "National Programming Contest 2024",
    "competition_level": "national",
    "rank": 1,
    "medal": "gold"
  },
  "tags": ["programming", "competition", "national"]
}
```

#### GET /api/achievements
Dapatkan semua prestasi user (requires authentication)

#### GET /api/achievements/references
Dapatkan referensi prestasi dengan status (requires authentication)

#### GET /api/achievements/:id
Dapatkan prestasi berdasarkan ID (requires authentication)

#### PUT /api/achievements/:id
Update prestasi (requires authentication)

#### DELETE /api/achievements/:id
Hapus prestasi (requires authentication)

#### POST /api/achievements/:achievement_id/submit
Submit prestasi untuk verifikasi (requires authentication)

### Verification (Lecturer Only)

#### GET /api/achievements/verify/pending
Dapatkan prestasi yang menunggu verifikasi (requires authentication)

#### POST /api/achievements/verify/:reference_id
Verifikasi prestasi (requires authentication)
```json
{
  "status": "verified",
  "rejection_note": "Optional rejection note if status is rejected"
}
```

## Data Models

### User
```json
{
  "id": "ObjectId",
  "username": "string",
  "email": "string",
  "password": "string (hashed)",
  "full_name": "string",
  "role_id": "ObjectId",
  "is_active": "boolean",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### Achievement
```json
{
  "id": "ObjectId",
  "student_id": "ObjectId",
  "object_id": "string",
  "student_info": "string",
  "category": "string",
  "title": "string",
  "description": "string",
  "details": {
    // Dynamic fields based on category
  },
  "custom_fields": [
    {
      "name": "string",
      "value": "any"
    }
  ],
  "attachments": [
    {
      "file_name": "string",
      "file_url": "string",
      "file_type": "string",
      "file_size": "number"
    }
  ],
  "tags": ["string"],
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

## Default Users

Setelah aplikasi pertama kali dijalankan, akan dibuat user admin default:
- **Username**: admin
- **Password**: admin123
- **Role**: admin

## Dummy Data

Untuk testing dan development, tersedia dummy data yang dapat di-seed:

### Dummy Users
- **3 Dosen**: dosen1, dosen2, dosen3 (password: admin123)
- **5 Mahasiswa**: mahasiswa1-5 (password: admin123)

### Dummy Achievements
- **5 Prestasi** dengan berbagai kategori (competition, publication, organization, certification)
- **Berbagai Status**: verified, submitted, rejected, draft
- **Field Dinamis**: Setiap kategori memiliki field khusus
- **Custom Fields**: Contoh penggunaan field tambahan
- **File Attachments**: Dummy file attachments

### Running Dummy Data Seeding
```bash
# Seed dummy data saja
go run cmd/seed/main.go -type=dummy
./scripts/seed.sh dummy

# Seed semua data (initial + dummy)
go run cmd/seed/main.go -type=all
./scripts/seed.sh all
```

Lihat [database/DUMMY_DATA.md](database/DUMMY_DATA.md) untuk detail lengkap dummy data.

## Architecture

Aplikasi ini dibangun tanpa menggunakan handler/controller pattern dan GORM sesuai requirement. Struktur yang digunakan:

- **Models**: Definisi struktur data
- **Repository**: Layer akses database langsung ke MongoDB
- **Service**: Business logic layer
- **Routes**: HTTP routing langsung tanpa controller
- **Middleware**: Authentication dan authorization

## Hybrid Database Architecture

Aplikasi ini menggunakan arsitektur hybrid database:

### PostgreSQL (Master Data)
- **Location**: `database/migrations/postgres/`
- **Format**: SQL files (.sql)
- **Data**: users, roles, permissions, students, lecturers
- **Why**: ACID compliance, relational integrity, mature ecosystem

### MongoDB (Achievement Data)
- **Location**: `database/migrations/mongo/`
- **Format**: JavaScript files (.js)  
- **Data**: achievements, achievement_references
- **Why**: Flexible schema, better performance untuk document data

Lihat [database/HYBRID_ARCHITECTURE.md](database/HYBRID_ARCHITECTURE.md) untuk detail lengkap.

### Migration Files Structure

**PostgreSQL (Master Data):**
```
database/migrations/postgres/
├── 001_create_users_table.sql
├── 002_create_roles_table.sql
├── 003_create_permissions_table.sql
├── 004_create_role_permissions_table.sql
├── 005_create_students_table.sql
├── 006_create_lecturers_table.sql
├── 009_seed_admin_user.sql
└── 010_seed_dummy_data.sql
```

**MongoDB (Achievement Data):**
```
database/migrations/mongo/
├── 001_create_achievements_collections.js
└── 005_seed_dummy_data.js
```

### Running Migrations

```bash
# PostgreSQL (Master Data)
go run cmd/migrate/main.go -db=postgres

# MongoDB (Achievement Data)
go run cmd/migrate/main.go -db=mongo

# Using scripts
./scripts/migrate.sh postgres  # Linux/Mac
./scripts/migrate.sh mongo     # Linux/Mac
scripts\migrate.bat postgres   # Windows
scripts\migrate.bat mongo      # Windows
```

## Development

Untuk development, pastikan database berjalan dan jalankan:

```bash
go run main.go
```

Aplikasi akan otomatis:
1. Connect ke PostgreSQL (master data)
2. Connect ke MongoDB (achievement data)
3. Run migrations untuk kedua database
4. Seed data awal (roles dan admin user)
5. Start HTTP server

## Production Deployment

1. Set environment variables yang sesuai
2. Gunakan JWT secret yang kuat
3. Setup MongoDB dengan proper security
4. Build aplikasi: `go build -o app main.go`
5. Run: `./app`