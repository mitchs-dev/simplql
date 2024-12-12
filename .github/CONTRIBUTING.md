# Contributing Guidelines

We welcome contributions to the project. Please follow the guidelines below.

## Reporting issues

If you encounter any issues with the project, please file a bug report in GitHub issues following the pre-defined template. 

If you face a security issue, **please do not report it in the public issue tracker**. Instead, please follow the instructions in the [Security Policy](./SECURITY.md).

### Experience required to work on this project

To work on this project, it's suggested to have some experience with the following technologies:

> **If you don't have experience with these technologies, we encourage you to still contribute and learn along the way!**

- [Go](https://golang.org/)
- [SQLite](https://www.sqlite.org/index.html) (or [SQL](https://en.wikipedia.org/wiki/SQL) in general)

There are many other technologies being used in this project, but the two above are the key ones. 

## Running the project locally

### Prerequisites

To run the project locally, you must have the following prerequisites installed:

- [Go 1.23](https://golang.org/dl/)
- [Git](https://git-scm.com/downloads)
- [Docker](https://www.docker.com/products/docker-desktop) (If you want to run the project in a container or are specifically working on containerization)

These tools aren't required, but they are recommended:

- [VSCodium](https://vscodium.com/) or [Visual Studio Code](https://code.visualstudio.com/)
- [yq](https://mikefarah.gitbook.io/yq/)
- [jq](https://stedolan.github.io/jq/)

### Project setup

To set up the project locally, follow these steps:

1. Clone the repository:

    ```bash
    # If you are using SSH
    git clone git@github.com:mitchs-dev/simplql.git 
    # If you are using HTTPS
    git clone https://github.com/mitchs-dev/simplql.git
    ```
2. Change into the project directory (`simplql`)
3. Install the project dependencies with `go mod tidy`
4. Ensure that you have a `config` file to customize any settings. You can copy from the default configuration under `app/pkg/configurationAndInitialization/configuration/default.yaml`
5. Change to the `app` directory
6. Run the project with `go run cmd/main.go -c <path to your config file>`


## Pull requests

We welcome pull requests. To contribute to the project, please follow these steps:

1. Fork the repository.
2. Create a new branch for your feature or bug fix. Ideally, name the branch after the issue you are addressing. For example, `issue-123`.
3. Make your changes in the new branch.
4. Test your changes.
5. Submit a pull request to the `main` branch of the repository.

Each pull request has a pre-defined template. Please fill out the template to the best of your ability.

## Documentation

As of right now, the project is in the early stages of development. We are working on improving the documentation. If you would like to contribute to the documentation, please keep reading.

## Becoming a contributor

Everyone who is a participant has a chance to become a contributor. To become a contributor, you must:

1. Follow the guidelines in this document.
2. Contribute to the project in a meaningful way, such as:
    - Submitting a pull request.
    - Reporting a bug.
    - Helping to improve the documentation.
    - Providing feedback on the project.
    - Helping to answer questions in the issue tracker.
3. Continue to make contributions to the project.

Overtime, the project team will review contributions made by participants. If the project team believes that a participant has made meaningful contributions to the project, they will be invited to become a contributor.

## Project meetings

As of right now, the project team is not holding regular meetings. However, we are considering holding meetings in the future. If you are interested in participating in project meetings, please let us know.