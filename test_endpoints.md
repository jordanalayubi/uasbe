# API Endpoint Testing Guide

## Authentication Setup
First, get JWT tokens for each role:

### 1. Admin Login
```bash
POST http://localhost:3000/api/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

### 2. Student Login
```bash
POST http://localhost:3000/api/auth/login
Content-Type: application/json

{
  "username": "mhs1",
  "password": "admin123"
}
```

### 3. Lecturer Login
```bash
POST http://localhost:3000/api/auth/login
Content-Type: application/json

{
  "username": "dosen1",
  "password": "admin123"
}
```

## FR-010: Admin View All Achievements

### Basic Request
```bash
GET http://localhost:3000/api/achievements/admin/all
Authorization: Bearer {{ADMIN_JWT_TOKEN}}
```

### With Filters
```bash
# Filter by status
GET http://localhost:3000/api/achievements/admin/all?status=submitted
Authorization: Bearer {{ADMIN_JWT_TOKEN}}

# Filter by category
GET http://localhost:3000/api/achievements/admin/all?category=competition
Authorization: Bearer {{ADMIN_JWT_TOKEN}}

# Filter by student
GET http://localhost:3000/api/achievements/admin/all?student_id={{STUDENT_USER_ID}}
Authorization: Bearer {{ADMIN_JWT_TOKEN}}

# Combined filters with pagination
GET http://localhost:3000/api/achievements/admin/all?status=verified&category=competition&sort_by=title&sort_order=asc&page=1&limit=5
Authorization: Bearer {{ADMIN_JWT_TOKEN}}
```

## FR-011: Achievement Statistics

### Student Statistics
```bash
# All time statistics
GET http://localhost:3000/api/achievements/statistics
Authorization: Bearer {{STUDENT_JWT_TOKEN}}

# Yearly statistics
GET http://localhost:3000/api/achievements/statistics?period=year&year=2024
Authorization: Bearer {{STUDENT_JWT_TOKEN}}

# Monthly statistics
GET http://localhost:3000/api/achievements/statistics?period=month&year=2024&month=12
Authorization: Bearer {{STUDENT_JWT_TOKEN}}
```

### Lecturer Statistics
```bash
# All time statistics for advisee
GET http://localhost:3000/api/achievements/statistics
Authorization: Bearer {{LECTURER_JWT_TOKEN}}

# Yearly statistics for advisee
GET http://localhost:3000/api/achievements/statistics?period=year&year=2024
Authorization: Bearer {{LECTURER_JWT_TOKEN}}
```

### Admin Statistics
```bash
# All time system statistics
GET http://localhost:3000/api/achievements/statistics
Authorization: Bearer {{ADMIN_JWT_TOKEN}}

# Yearly system statistics
GET http://localhost:3000/api/achievements/statistics?period=year&year=2024
Authorization: Bearer {{ADMIN_JWT_TOKEN}}

# Monthly system statistics
GET http://localhost:3000/api/achievements/statistics?period=month&year=2024&month=12
Authorization: Bearer {{ADMIN_JWT_TOKEN}}
```

## Expected Response Format

### Admin All Achievements Response
```json
{
  "success": true,
  "message": "All achievements retrieved successfully",
  "summary": {
    "total_achievements": 10,
    "filtered_results": 5,
    "page_results": 5,
    "filters_applied": {
      "status": "submitted",
      "category": "",
      "student_id": ""
    }
  },
  "data": [
    {
      "achievement": {
        "id": "achievement_id",
        "title": "Achievement Title",
        "category": "competition",
        "description": "Achievement description"
      },
      "reference": {
        "id": "reference_id",
        "status": "submitted",
        "submitted_at": "2024-12-19T00:00:00Z",
        "created_at": "2024-12-19T00:00:00Z"
      },
      "student_info": {
        "student_id": "12345678",
        "program_study": "D4 Teknik Informatika",
        "academic_year": "2024",
        "user_id": "student_user_id"
      }
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_items": 10,
    "total_pages": 1,
    "has_next_page": false,
    "has_previous_page": false
  }
}
```

### Statistics Response Format
```json
{
  "success": true,
  "message": "Achievement statistics retrieved successfully",
  "code": "STATISTICS_SUCCESS",
  "data": {
    "overview": {
      "total_achievements": 5,
      "verified_achievements": 3,
      "pending_achievements": 1,
      "draft_achievements": 1
    },
    "by_category": {
      "competition": 3,
      "research": 1,
      "community_service": 1
    },
    "by_status": {
      "verified": 3,
      "submitted": 1,
      "draft": 1
    },
    "by_competition_level": {
      "national": 2,
      "regional": 1,
      "international": 1
    }
  },
  "user_info": {
    "user_id": "user_id",
    "username": "username",
    "role": "student"
  },
  "query_parameters": {
    "period": "all",
    "year": 2024,
    "month": 0
  },
  "timestamp": "2024-12-19T00:00:00Z"
}
```

## Testing Steps

1. **Start the application**: `go run main.go`
2. **Login with each role** to get JWT tokens
3. **Test admin endpoints** with admin token
4. **Test statistics endpoints** with each role token
5. **Verify response formats** match expected structure
6. **Test error cases** (invalid tokens, wrong roles, etc.)

## Common Issues to Check

1. **Authentication**: Ensure JWT tokens are valid and not expired
2. **Authorization**: Verify role-based access control works
3. **Data consistency**: Check that student IDs and achievement IDs match
4. **Pagination**: Test different page sizes and page numbers
5. **Filters**: Verify all filter combinations work correctly
6. **Empty results**: Test behavior when no data matches filters