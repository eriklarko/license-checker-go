.PHONY: lint typecheck test

install-poetry:
	@echo "Installing Poetry..."
	@if ! command -v poetry &> /dev/null; then \
		curl -sSL https://install.python-poetry.org | python3 -; \
	else \
		echo "Poetry is already installed."; \
	fi


lint:
	poetry run black .

typecheck:
	poetry run mypy .

test:
	poetry run python go_mod_test.py
