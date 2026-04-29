[English](./README.md) | [简体中文](./README.zh.md)

# Project Overview

`goark` is a Go backend engineering practice project based on [Gin](https://github.com/gin-gonic/gin). It provides a layered, maintainable, and scalable service structure with multiple app modules.

---

# Features

* **Clear Project Structure**: Inspired by [project-layout](https://github.com/golang-standards/project-layout), follows layered architecture principles, organized for team collaboration and long-term maintenance.
* **Common Component Integration**: Includes built-in examples for MySQL, Redis, and Elasticsearch.
* **Full Link Logging**: Provides a custom logging package `glog` based on `zap`, supporting full trace ID propagation across MySQL, Redis, ES, and HTTP calls.
* **Code Generation Tool**: Comes with a command-line tool `gocli` that can generate standardized code (including model, dao, object, dto, code, service, controller, router layers) based on config.
* **Swagger API Documentation**: Automatically generate interactive API docs using `swaggo` for easier frontend-backend collaboration and testing.
* **Docker Support**: Includes a basic `Dockerfile` for containerized deployment.
* **Makefile Toolchain**: Provides a rich set of make commands to simplify code build, run, generation, Swagger docs, and Docker deployment.
* **Growing Golib Library**: Common utility components are abstracted and reusable via the [golib](https://github.com/morehao/golib) package.

---

# Project Structure

Follows [project-layout](https://github.com/golang-standards/project-layout). Current structure:

```bash
.
├── apps
│   ├── demo
│   │   ├── cmd
│   │   ├── client
│   │   ├── config
│   │   ├── dao
│   │   ├── model
│   │   ├── docs
│   │   ├── internal
│   │   │   ├── controller
│   │   │   ├── dto
│   │   │   ├── router
│   │   │   └── service
│   │   ├── middleware
│   │   ├── object
│   │   └── scripts
│   └── iam
│       ├── cmd
│       ├── config
│       ├── docs
│       ├── dao
│       ├── model
│       ├── internal
│       │   ├── controller
│       │   ├── dto
│       │   ├── router
│       │   └── service
│       ├── object
│       └── scripts
├── pkg
│   ├── code
│   ├── dbclient
│   ├── testsetup
│   └── utils
└── scripts
```

---

# Core Features

## Code Generation

Install the CLI tool:

```bash
go install github.com/morehao/gocli@latest
```

Ensure a `code_gen.yaml` config file exists under the application directory, e.g., `goark/apps/demo/config/code_gen.yaml`.

Run code generation commands:

```bash
# Generate full module based on table
make codegen APP=demo COMMAND=module

# Generate only model code
make codegen APP=demo COMMAND=model

# Generate API endpoint code
make codegen APP=demo COMMAND=api
```

See [generate](https://github.com/morehao/gocli?tab=readme-ov-file#generate) for full documentation.

---

## API Documentation

Install Swagger tool:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Generate Swagger docs:

```bash
make swag APP=demo
```

Access docs at (dev mode):

```
http://localhost:8099/demo/redocs
```

---

## Project Deployment

Build Docker image:

```bash
make docker-build APP=demo
```

Run container:

```bash
make docker-run APP=demo
```

---

## Quickly Scaffold a New Project

Install the `cutter` tool:

```bash
go install github.com/morehao/gocli@latest
```

Run under **the root of the template project (e.g., `./`)**:

```bash
gocli cutter -d /goProject/yourAppName
```

This will scaffold a new project named `yourAppName` under `/goProject` based on the current template.

See [cutter](https://github.com/morehao/gocli) for more usage details.

---

## Related Libraries

All related components are implemented in the [golib](https://github.com/morehao/golib) package.
