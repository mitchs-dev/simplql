# simplQL: A Lightweight SQLite Server

> This project is still **under development**. Some aspects of this project still need to be implemented and tested. Please refer to the [Development and Roadmap](#development-and-roadmap) section for more information.

## Items to Phase 1 completion

- [x] Database generation
- [x] RESTful API
- [x] JSON responses
- [x] CRUD operations for DB
- [x] CRUD operations for Authentication
- [x] Login/Logout mechanisms
- [x] JWT token generation
- [ ] RBAC (DB-level)
- [ ] Ensure encryption operations are stable
- [ ] Provide install instructions
- [ ] Create deployment packaging
- [ ] Ensure that project meets ACID compliance
- [ ] Ensure that project statements are parameterized
- [ ] Transaction management

## Introduction

`simplQL` is a lightweight SQLite server designed to provide a simple and efficient solution for applications that require a basic database functionality. Unlike traditional SQL servers, `simplQL` is focused on delivering a streamlined set of features tailored for specific use cases, ensuring a balance between simplicity and functionality.

> **Disclaimer**: `simplQL` is not intended to be a drop-in replacement for full-fledged SQL servers like MySQL or PostgreSQL. Instead, it is designed to cater to applications that need a lightweight, self-contained database solution with minimal setup and configuration.

## Key Features

- **ACID Compliance**: simplQL ensures Atomicity, Consistency, Isolation, and Durability for all database operations, providing a reliable and robust data storage solution.
- **Simple CRUD Operations**: simplQL offers a straightforward API for performing basic Create, Read, Update, and Delete operations on the database.
- **RESTful API**: The project exposes a RESTful API, allowing seamless integration with various client applications and frameworks.
- **JSON Responses**: All responses from the simplQL server are returned in a standardized JSON format, making it easy to parse and consume the data.
- **Authentication and Authorization**: simplQL supports per-database user management and role-based access control (RBAC), ensuring secure access to the data.
- **Static Database and Tables**: The database structure, including tables and their schemas, is defined in a configuration file and initialized during startup, providing a predictable and maintainable setup.
- **Database Versioning and Migrations**: simplQL includes a versioning system that allows for easy database schema updates and migrations, simplifying the management of database changes over time.
- **Per-Database User Isolation**: Each database in simplQL has its own set of users, ensuring complete isolation and security between different data stores.
- **Per-Database RBAC**: The RBAC system in simplQL is scoped to individual databases, allowing for granular control over user permissions and access rights.
- **Per-entry encryption**: SimplQL can be configured to encrypt each entry of every database, ensuring that data is secure at rest.
  > **Important**: To validate the encryption key; SimplQL uses the `id` command to generate a file. If your system user's id or groups have changed, the validation will fail.

## Use Cases

simplQL is designed to cater to the following use cases:

- **Single Application Database**: simplQL is an excellent fit for applications that require a simple, self-contained database to store and manage their data.
- **Moving Data off the Local Filesystem**: If you need to transition from storing data in local files to a more structured database solution, simplQL provides a lightweight and easy-to-use option.
- **API-driven Applications**: The RESTful API and JSON responses of simplQL make it a suitable choice for API-driven applications that need a simple, programmatic database interface.

## Architecture and Design

The simplQL server is designed with a modular and extensible architecture, ensuring that the core functionality remains focused and lightweight, while allowing for the addition of optional features or plugins if required.

The core of the simplQL server consists of the following components:

1. **API Layer**: Responsible for handling incoming requests, parsing parameters, and translating them into database operations.
2. **Database Layer**: Manages the SQLite database, including CRUD operations, schema management, and data persistence.
3. **Authentication and Authorization Layer**: Handles user management, authentication, and role-based access control.
4. **Configuration and Initialization Layer**: Responsible for loading the database and table definitions from the configuration file and setting up the initial database state.
  
The modular design of simplQL allows for the addition of optional features or extensions, such as:

- **Advanced Query Capabilities**: Expanding the query language and functionality beyond the basic CRUD operations.
- **Backup and Restore**: Implementing mechanisms for database backup and restoration.
- **Monitoring and Logging**: Integrating monitoring and logging capabilities to aid in troubleshooting and performance analysis.
- **Clustering and High Availability**: Exploring options for distributed or replicated database setups, if required by the target use cases.

## Development and Roadmap

The simplQL project is currently in the early stages of development, with the core functionality and targeted use cases defined. The development roadmap includes the following milestones:

- **Phase 1 (Current)**: Implement the core features, including simple CRUD operations, RESTful API, JSON responses, authentication, and static database/table management.
- **Phase 2**: Implement HA and clustering capabilities, allowing for replication and distribution of databases across multiple nodes.
- **Phase 3**: Implement per-database user isolation and RBAC, enhancing the security and access control features.
- **Phase 4**: Implement database migration and fix SELECT transaction logic.


### Unplanned phase features

The following tasks are planned for future phases but are not currently in the development pipeline:
- Implement SELECT query transaction logic

### Potential Future Features

Here are some additional features which are up for consideration in future phases:
- Encryption key rotation
- Remote databases (using simplQL as API front-end and query executor)
- External transaction tracking
- Backup and restore
- Monitoring exposure

Throughout the development process, the simplQL team will maintain a strong focus on simplicity, ease of use, and alignment