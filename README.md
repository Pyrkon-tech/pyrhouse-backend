# Go Test Database

A warehouse inventory management application with Google Sheets integration.

## 🚀 Features

- Asset management
- Item categories management
- Stock management
- Location transfers
- Google Sheets integration
- Role-based access control
- Audit logging
- Service desk functionality

## 🛠️ Technologies

- **Backend**: Go 1.21+
- **Database**: PostgreSQL 15
- **Migrations**: golang-migrate
- **Containerization**: Docker & Docker Compose
- **CI/CD**: GitHub Actions
- **API Documentation**: OpenAPI 3.1.0 with Redoc

## 📋 Requirements

- Go 1.21+
- PostgreSQL 15+
- Docker & Docker Compose

## 🚀 Quick Start

### With Docker Compose

```bash
# Clone repository
git clone <repository-url>
cd go-test-db

# Start with Docker Compose
docker-compose up -d

# Application will be available at http://localhost:8080
```

### Locally

```bash
# Install dependencies
go mod download

# Configure database
export DATABASE_URL="postgres://username:password@localhost:5432/dbname?sslmode=disable"

# Run migrations
go run cmd/migrate/main.go

# Start application
go run main.go
```

## 🔧 Configuration

### Environment Variables

```bash
DATABASE_URL=postgres://username:password@localhost:5432/dbname?sslmode=disable
GOOGLE_CREDENTIALS_FILE=path/to/credentials.json
```

### Google Sheets API

1. Create a project in Google Cloud Console
2. Enable Google Sheets API
3. Create a service account and download credentials.json
4. Place the file in `configs/` and set the path in environment variables

## 📚 API Documentation

API documentation is automatically generated and published to GitHub Pages when the OpenAPI specification is updated:

- **Documentation**: Available at `https://{username}.github.io/{repository-name}/`
- **OpenAPI Spec**: Located at `docs/openapi.yaml`
- **Generation**: Uses Redoc CLI to create beautiful, interactive documentation

The documentation is automatically updated when changes are pushed to the `main` branch.

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test
go test ./internal/inventory/assets/...
```

## 📊 CI Pipeline

The project uses GitHub Actions for automation:

### Workflows

1. **CI Pipeline** (`.github/workflows/ci.yml`)
   - Unit and integration tests
   - Security analysis (Trivy, govulncheck)
   - Docker image building
   - Code coverage reporting

2. **API Documentation** (`.github/workflows/docs.yml`)
   - Generates API documentation using Redoc CLI
   - Publishes to GitHub Pages
   - Triggers on changes to OpenAPI specification

3. **Dependency Review** (`.github/workflows/dependency-review.yml`)
   - Automatic dependency checking in PRs

4. **CodeQL** (`.github/workflows/codeql.yml`)
   - Advanced code security analysis

### Dependabot

Automatic dependency updates:
- Go modules (weekly)
- GitHub Actions (weekly)
- Docker images (weekly)

## 📁 Project Structure

```
├── cmd/                    # Application entry points
│   ├── dev/               # Development server
│   ├── migrate/           # Migration tool
│   └── server/            # Production server
├── docs/                  # Documentation
│   ├── openapi.yaml       # OpenAPI specification
│   └── index.html         # Generated API documentation
├── internal/              # Internal application code
│   ├── auditlog/          # Audit logs
│   ├── database/          # Database configuration
│   ├── di/                # Dependency injection
│   ├── inventory/         # Inventory module
│   ├── integrations/      # External integrations
│   ├── locations/         # Location management
│   ├── logging/           # Logging system
│   ├── middleware/        # HTTP middleware
│   ├── models/            # Data models
│   ├── repository/        # Data access layer
│   ├── roles/             # Role system
│   ├── routing/           # HTTP routing
│   ├── security/          # Security and authorization
│   ├── service_desk/      # Service desk
│   └── users/             # User management
├── migrations/            # Database migrations
├── postgres/              # PostgreSQL configuration
└── .github/               # GitHub Actions configuration
```

## 🔒 Security

- Dependency vulnerability analysis (Dependabot)
- Code scanning (CodeQL)
- Docker image scanning (Trivy)
- Audit logging for all operations
- Role-based access control

## 📈 Monitoring

- Health check endpoint: `/health`
- Application metrics
- Structured logging
- Audit trail

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Standards

- Use `gofmt` for formatting
- Add tests for new features
- Update documentation
- Use conventional commits

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0) **with an additional restriction** that prohibits offering it as a hosted service (SaaS) to third parties.

### ✅ What you can do

- Self-host the app for your own use (even commercially)
- Modify and use the code internally
- Contribute to the open-source core

### ❌ What you may not do

- Offer this software or a modified version as a publicly accessible hosted service (SaaS) without a commercial license.

### Commercial Offerings

We offer hosted services, premium plugins, and commercial licenses. Contact us at [warrmag7@gmail.com] for details.

## 🆘 Support

- Issues: [GitHub Issues](https://github.com/your-repo/issues)
- Documentation: [Wiki](https://github.com/your-repo/wiki)
- API Docs: [GitHub Pages](https://pyrkon-tech.github.io/pyrhouse-backend/)
- Email: warrmag7@gmail.com