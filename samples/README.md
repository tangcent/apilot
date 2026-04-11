# APilot Integration Tests

This directory contains sample projects for all supported frameworks, used for integration testing APilot's functionality.

## Structure

Each subdirectory represents a sample project for a specific framework:

- **Go**: `go-gin`, `go-echo`, `go-fiber`
- **Java**: `java-springmvc`, `java-jaxrs`, `java-feign`
- **Node.js**: `node-express`, `node-fastify`, `node-nestjs`
- **Python**: `python-fastapi`, `python-flask`, `python-django`

## Running Tests

### Run all integration tests

```bash
./run_all_tests.sh
```

### Run tests for a specific framework

```bash
cd go-gin
./test.sh
```

## Test Process

Each test script:

1. Uses APilot to export APIs from the sample project to Markdown format
2. Verifies the output contains expected endpoints
3. Returns exit code 0 on success, non-zero on failure

## Adding New Frameworks

To add a new framework:

1. Create a new directory: `samples/<language>-<framework>`
2. Add sample code demonstrating API endpoints
3. Copy `test_template.sh` to the directory as `test.sh`
4. Make the test script executable: `chmod +x test.sh`
5. Update this README with the new framework

## CI Integration

These integration tests are automatically run in the CI pipeline after unit tests. See `.github/workflows/ci.yml` for details.
