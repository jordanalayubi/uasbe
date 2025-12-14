# Design Document

## Overview

The Student Achievement Management System is a REST API backend built with Go using the Fiber web framework. The system implements a clean architecture pattern with clear separation between handlers, services, repositories, and models. It provides comprehensive functionality for managing student achievements with role-based access control, approval workflows, and flexible data structures.

The system supports three primary user roles: Students who can create and manage their achievements, Lecturers who verify achievements of students under their supervision, and Administrators who have full system access. The architecture emphasizes security through JWT-based authentication, data integrity through proper validation, and scalability through modular design patterns.

## Architecture

The system follows a layered architecture pattern with clear separation of concerns:

### Layer Structure
- **Handler Layer**: HTTP request/response handling and routing
- **Service Layer**: Business logic and workflow orchestration  
- **Repository Layer**: Data access and persistence operations
- **Model Layer**: Data structures and domain entities
- **Middleware Layer**: Cross-cutting concerns (authentication, logging, CORS)

### Technology Stack
- **Framework**: Go Fiber v2 for high-performance HTTP handling
- **Database**: PostgreSQL with GORM ORM for data persistence
- **Authentication**: JWT tokens with role-based permissions
- **File Storage**: Local filesystem for achievement attachments
- **Validation**: Built-in Go validation with custom business rules

### Deployment Architecture
The system is designed as a stateless REST API that can be horizontally scaled. Database connections are managed through GORM connection pooling, and file uploads are handled with configurable size limits.

## Components and Interfaces

### Core Components

#### Authentication Service
- **Purpose**: Manages user authentication and JWT token generation
- **Key Methods**: 
  - `Login(credentials)` - Validates credentials and returns JWT
  - `ValidateToken(token)` - Verifies JWT and extracts user claims
  - `RefreshToken(token)` - Generates new token for authenticated users

#### Achievement Service  
- **Purpose**: Orchestrates achievement CRUD operations and workflow
- **Key Methods**:
  - `CreateAchievement(data, studentID)` - Creates new achievement record
  - `UpdateAchievement(id, data, userID)` - Updates existing achievement
  - `SubmitForVerification(id, studentID)` - Changes status to submitted
  - `VerifyAchievement(id, lecturerID, decision)` - Approves/rejects achievement

#### User Management Service
- **Purpose**: Handles user account operations and role management
- **Key Methods**:
  - `CreateUser(userData)` - Creates new user account with role
  - `UpdateUserRole(userID, newRole)` - Modifies user permissions
  - `GetUsersByRole(role)` - Retrieves users filtered by role

### Interface Contracts

#### Repository Interfaces
```go
type AchievementRepository interface {
    Create(achievement *Achievement) error
    GetByID(id uint) (*Achievement, error)
    GetByStudentID(studentID uint) ([]Achievement, error)
    Update(achievement *Achievement) error
    Delete(id uint) error
}

type UserRepository interface {
    Create(user *User) error
    GetByUsername(username string) (*User, error)
    GetByID(id uint) (*User, error)
    Update(user *User) error
}
```

#### Service Interfaces
```go
type AuthService interface {
    Login(username, password string) (*AuthResponse, error)
    ValidateToken(token string) (*UserClaims, error)
}

type AchievementService interface {
    CreateAchievement(req *CreateAchievementRequest, studentID uint) error
    VerifyAchievement(id uint, lecturerID uint, approved bool, note string) error
}
```

## Data Models

### User Entity
```go
type User struct {
    ID        uint      `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    Password  string    `json:"-"`
    Role      string    `json:"role"` // Admin, Student, Lecturer
    IsActive  bool      `json:"is_active"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### Achievement Entity
```go
type Achievement struct {
    ID            uint                `json:"id"`
    StudentID     uint                `json:"student_id"`
    Title         string              `json:"title"`
    Category      string              `json:"category"` // Academic, Non-Academic
    Level         string              `json:"level"`    // Local, Regional, National, International
    Rank          string              `json:"rank"`
    EventDate     time.Time           `json:"event_date"`
    Organizer     string              `json:"organizer"`
    Location      string              `json:"location"`
    Description   string              `json:"description"`
    Points        int                 `json:"points"`
    Status        string              `json:"status"` // Pending, Verified, Rejected
    VerifiedBy    *uint               `json:"verified_by"`
    VerifiedAt    *time.Time          `json:"verified_at"`
    RejectionNote string              `json:"rejection_note"`
    Attachments   []Attachment        `json:"attachments"`
    CreatedAt     time.Time           `json:"created_at"`
    UpdatedAt     time.Time           `json:"updated_at"`
}
```

### Student Entity
```go
type Student struct {
    ID        uint      `json:"id"`
    UserID    uint      `json:"user_id"`
    NIM       string    `json:"nim"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Program   string    `json:"program"`
    Angkatan  int       `json:"angkatan"`
    AdvisorID *uint     `json:"advisor_id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### Lecturer Entity
```go
type Lecturer struct {
    ID         uint      `json:"id"`
    UserID     uint      `json:"user_id"`
    NIDN       string    `json:"nidn"`
    Name       string    `json:"name"`
    Email      string    `json:"email"`
    Department string    `json:"department"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}
```

### Attachment Entity
```go
type Attachment struct {
    ID            uint      `json:"id"`
    AchievementID uint      `json:"achievement_id"`
    FileName      string    `json:"file_name"`
    FilePath      string    `json:"file_path"`
    FileType      string    `json:"file_type"`
    FileSize      int64     `json:"file_size"`
    CreatedAt     time.Time `json:"created_at"`
}
```

### Achievement History Entity
```go
type AchievementHistory struct {
    ID            uint      `json:"id"`
    AchievementID uint      `json:"achievement_id"`
    Status        string    `json:"status"`
    Note          string    `json:"note"`
    ChangedBy     uint      `json:"changed_by"`
    CreatedAt     time.Time `json:"created_at"`
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Authentication and Authorization Properties

**Property 1: Valid authentication generates tokens**
*For any* valid user credentials, authentication should generate a JWT token containing the correct role permissions for that user
**Validates: Requirements 1.1**

**Property 2: Invalid credentials are rejected**
*For any* invalid credential combination (wrong username, wrong password, non-existent user), the authentication system should reject the request and return an appropriate error message
**Validates: Requirements 1.2**

**Property 3: Token validation enforces permissions**
*For any* valid JWT token and protected endpoint, the system should only allow access if the token's role has permission for that specific operation
**Validates: Requirements 1.3**

**Property 4: Expired tokens are rejected**
*For any* expired JWT token, all protected endpoints should reject the request and require re-authentication
**Validates: Requirements 1.4**

**Property 5: Role boundaries are enforced**
*For any* user and system operation, the user should only be able to perform operations that are explicitly allowed for their role
**Validates: Requirements 1.5**

### Achievement Management Properties

**Property 6: Achievement creation stores complete data**
*For any* valid achievement data submitted by a student, the system should store all required fields and make them retrievable
**Validates: Requirements 2.1**

**Property 7: Achievement updates preserve immutable fields**
*For any* achievement update operation, the creation timestamp and student ownership should remain unchanged while allowing modification of editable fields
**Validates: Requirements 2.2**

**Property 8: Students see only their own achievements**
*For any* student user, querying achievements should return only records where the student is the owner, never achievements belonging to other students
**Validates: Requirements 2.3**

**Property 9: Achievement deletion removes records**
*For any* achievement deleted by its owner, subsequent queries for that achievement should return not found
**Validates: Requirements 2.4**

**Property 10: Submission changes status correctly**
*For any* achievement submitted for verification, the status should change to "submitted" and the assigned lecturer should be notified
**Validates: Requirements 2.5**

### Approval Workflow Properties

**Property 11: Lecturers see only supervised achievements**
*For any* lecturer, querying pending achievements should return only achievements from students under their supervision
**Validates: Requirements 3.1**

**Property 12: Approval updates status and timestamp**
*For any* achievement approved by a lecturer, the status should change to "verified" and the verification timestamp should be recorded
**Validates: Requirements 3.2**

**Property 13: Rejection requires documentation**
*For any* achievement rejection, the system should require a rejection note and update the status to "rejected"
**Validates: Requirements 3.3**

**Property 14: Feedback is stored and linked**
*For any* feedback provided by a lecturer, it should be stored and properly associated with the corresponding achievement
**Validates: Requirements 3.4**

**Property 15: Review displays complete information**
*For any* achievement under review, all relevant details and supporting documentation should be available to the reviewer
**Validates: Requirements 3.5**

### Administrative Properties

**Property 16: User creation assigns correct roles**
*For any* new user account created by an administrator, the user should be assigned the specified role and have secure credentials generated
**Validates: Requirements 4.1**

**Property 17: Admin reports provide accurate data**
*For any* system report requested by an administrator, the data should accurately reflect the current state of achievements and user activities
**Validates: Requirements 4.3**

**Property 18: Admin has full CRUD access**
*For any* achievement record and administrative user, all CRUD operations should be permitted regardless of ownership
**Validates: Requirements 4.4**

### Data Structure Properties

**Property 19: Achievement types support required fields**
*For any* achievement type (competition, publication, certification, organizational), the system should store all type-specific required fields
**Validates: Requirements 5.1**

**Property 20: Competition achievements store complete data**
*For any* competition achievement, the system should store competition name, level, rank, medal type, event date, and organizer information
**Validates: Requirements 5.2**

**Property 21: Publication achievements store complete data**
*For any* publication achievement, the system should store publication type, journal name, ISBN/ISSN, publisher, and issue details
**Validates: Requirements 5.3**

**Property 22: Certification achievements store complete data**
*For any* certification achievement, the system should store certification name, issuing body, certification number, validity period, and verification details
**Validates: Requirements 5.4**

**Property 23: Organizational achievements store complete data**
*For any* organizational achievement, the system should store organization name, position held, period of service, and activity scope
**Validates: Requirements 5.5**

### Data Integrity Properties

**Property 24: Invalid data is rejected with specific errors**
*For any* invalid input data, the system should validate against defined schemas and return specific, actionable error messages
**Validates: Requirements 6.1**

**Property 25: Error logging captures debugging information**
*For any* system error, appropriate error information should be logged with sufficient detail for debugging and monitoring
**Validates: Requirements 6.4**

**Property 26: HTTP responses follow standard format**
*For any* API request, the system should return appropriate HTTP status codes and consistently structured responses
**Validates: Requirements 6.5**

### API Endpoint Properties

**Property 27: User management endpoints provide complete operations**
*For any* user management requirement, the API should provide endpoints for authentication, user creation, role assignment, and profile management
**Validates: Requirements 7.1**

**Property 28: Achievement endpoints support role-filtered CRUD**
*For any* achievement operation, the API should provide CRUD functionality with appropriate role-based access filtering
**Validates: Requirements 7.2**

**Property 29: Workflow endpoints support complete approval process**
*For any* approval workflow step, the API should provide endpoints for submission, verification, rejection, and status tracking
**Validates: Requirements 7.3**

**Property 30: Reporting endpoints respect user permissions**
*For any* reporting request, the API should provide analytics and summary data filtered according to the requesting user's role and permissions
**Validates: Requirements 7.4**

## Error Handling

### Error Categories

#### Authentication Errors
- **Invalid Credentials**: Return 401 Unauthorized with clear message
- **Token Expired**: Return 401 Unauthorized requiring re-authentication
- **Insufficient Permissions**: Return 403 Forbidden with role requirements

#### Validation Errors
- **Missing Required Fields**: Return 400 Bad Request with field-specific messages
- **Invalid Data Format**: Return 400 Bad Request with format requirements
- **Business Rule Violations**: Return 422 Unprocessable Entity with rule explanation

#### Resource Errors
- **Not Found**: Return 404 Not Found for non-existent resources
- **Conflict**: Return 409 Conflict for duplicate or conflicting operations
- **Gone**: Return 410 Gone for soft-deleted resources

#### System Errors
- **Database Errors**: Return 500 Internal Server Error with logged details
- **File Upload Errors**: Return 413 Payload Too Large or 415 Unsupported Media Type
- **Rate Limiting**: Return 429 Too Many Requests with retry information

### Error Response Format
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data provided",
    "details": [
      {
        "field": "title",
        "message": "Title is required and cannot be empty"
      }
    ],
    "timestamp": "2024-12-14T10:30:00Z",
    "request_id": "req_123456789"
  }
}
```

### Error Handling Strategy
- **Graceful Degradation**: System continues operating when non-critical components fail
- **Transaction Rollback**: Database operations are rolled back on errors to maintain consistency
- **Logging**: All errors are logged with appropriate severity levels and context
- **User Feedback**: Clear, actionable error messages help users understand and resolve issues

## Testing Strategy

### Dual Testing Approach

The system will implement both unit testing and property-based testing to ensure comprehensive coverage and correctness verification.

#### Unit Testing
Unit tests will verify specific examples, edge cases, and integration points:
- **Authentication flows** with valid and invalid credentials
- **Role-based access control** for specific user-endpoint combinations  
- **Achievement CRUD operations** with sample data
- **Workflow state transitions** for approval processes
- **File upload handling** with various file types and sizes
- **Error scenarios** with specific invalid inputs

#### Property-Based Testing
Property-based tests will verify universal properties across all inputs using **Testify** and **go-fuzz** libraries:
- **Minimum 100 iterations** per property test to ensure statistical confidence
- **Smart generators** that create realistic test data within valid input spaces
- **Invariant verification** across all generated inputs
- Each property-based test will be tagged with: **Feature: student-achievement-management, Property {number}: {property_text}**

#### Testing Requirements
- Property-based tests must implement exactly one correctness property each
- Tests should avoid mocking when possible to validate real functionality  
- All property tests must reference their corresponding design document property
- Test failures must be investigated and either fixed in code or refined in specification
- Both unit and property tests are required for comprehensive validation

#### Test Organization
- Tests will be co-located with source files using `_test.go` suffix
- Property tests will be grouped by functional area (auth, achievements, workflow)
- Integration tests will verify end-to-end workflows across multiple components
- Performance tests will validate system behavior under load conditions