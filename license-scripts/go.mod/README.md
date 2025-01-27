# Go Module License Checker

This project provides a script to output the licenses for all dependencies of a Go project using the `go-licenses` tool.

## Setup

1. Install Poetry:
   
   Poetry is used for managing Python dependencies. To install Poetry, run:

   ```
   make install-poetry
   ```

   For more installation options, visit the [Poetry documentation](https://python-poetry.org/docs/#installation).

2. Install project dependencies:

   ```
   poetry install
   ```

## Usage

### Running the script

To run the `go_mod.py` script:


## Development

This project uses a Makefile to manage development tasks. Here's how to run tests, lint, and typecheck:

#### Running Tests

To run the tests for this project:  

```
make test
```

#### Running Lint

To run the linter for this project:

```
make lint
```

#### Running Typecheck

To run the typechecker for this project:

```
make typecheck
```

#### Running All

To run the linter, typechecker, and tests for this project:

```
make all
```

