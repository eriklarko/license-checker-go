#!/usr/bin/env python3

"""
This script outputs the licenses for all dependencies of a Go project.

It uses the go-licenses tool to generate a report of dependencies and their licenses.
The script can be configured to use a specific path for the go-licenses tool and the go.mod file.

Usage:
    python3 go.mod.py [--go-licenses-path PATH] [--go-mod-file PATH] [--no-install]

Example:
    python3 go.mod.py --go-licenses-path /path/to/go-licenses --go-mod-file /path/to/go.mod

The script will install go-licenses if it's not found in the specified path, unless --no-install is specified.
In CI environments, --no-install is the default behavior. (Defined by the
precense of the "CI" environment variable)
"""

import os
import re
import shutil
import sys
import subprocess
import argparse
from pathlib import Path


def parse_arguments() -> argparse.Namespace:
    """
    Parse command-line arguments.

    Returns:
        argparse.Namespace: Parsed arguments
    """
    parser = argparse.ArgumentParser(
        description="Output licenses for all dependencies."
    )
    parser.add_argument(
        "--go-licenses-path", default="go-licenses", help="Path to go-licenses tool"
    )
    parser.add_argument("--go-mod-file", default="./go.mod", help="Path to go.mod file")
    parser.add_argument(
        "--no-install",
        action="store_true",
        help="Do not attempt to install go-licenses if not found",
        default=False if os.environ.get("CI") is None else True
    )

    return parser.parse_args()


def assert_go_mod_file_exists(go_mod_file: str) -> None:
    """
    Check if the specified go.mod file exists.

    Args:
        go_mod_file (str): Path to the go.mod file

    Raises:
        SystemExit: If the go.mod file is not found
    """
    if not os.path.isfile(go_mod_file):
        go_mod_file_abs = Path(go_mod_file).resolve()
        print(f"Error: {go_mod_file_abs} file not found", file=sys.stderr)
        sys.exit(1)


def install_go_licenses() -> None:
    """
    Install go-licenses if it's not found in the specified path.
    """
    subprocess.run(
        ["go", "install", "github.com/google/go-licenses@latest"], check=True
    )


def get_package_name(go_mod_file: str) -> str:
    """
    Extract the package name from the go.mod file.

    Args:
        go_mod_file (str): Path to the go.mod file

    Returns:
        str: Package name
    """
    with open(go_mod_file, "r") as f:
        first_line = f.readline().strip()
    return first_line.split()[1]


def get_dependencies(go_mod_file: str) -> list[str]:
    """
    Get a list of dependencies from the go.mod file.

    Args:
        go_mod_file (str): Path to the go.mod file

    Returns:
        list[str]: List of dependencies
    """
    dependencies = []
    with open(go_mod_file, "r") as f:
        for line in f:
            # dependenceis are listed either with
            #  require dep 
            # or
            #  require (
            #      dep
            #  )

            is_single_line_require = line.startswith("require") and not line.startswith("require (")
            is_part_of_multi_line_require = re.match(r'^\s+\S', line) # matches any whitespace followed by a non-whitespace character
            if is_single_line_require or is_part_of_multi_line_require:
                parts = line.split()
                if len(parts) >= 2:
                    dependencies.append(parts[0])
                else:
                    raise ValueError(f"Unexpected format in go.mod file: {line}")

    return dependencies


def run_go_licenses(go_licenses_path: str, package: str) -> str:
    """
    Run go-licenses to generate a report for the specified package.

    Args:
        go_licenses_path (str): Path to the go-licenses tool
        package (str): Package name

    Returns:
        str: Output of go-licenses report
    """
    result = subprocess.run(
        [go_licenses_path, "report", package],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        print(result.stderr, file=sys.stderr)
        sys.exit(result.returncode)

    return result.stdout


def main() -> None:
    args = parse_arguments()

    assert_go_mod_file_exists(args.go_mod_file)

    # Check if the tool is in the PATH or at the specified location
    if not shutil.which(args.go_licenses_path):
        if args.no_install:
            print(f"Error: {args.go_licenses_path} not found and installation is disabled.", file=sys.stderr)
            sys.exit(1)
        else:
            print(f"{args.go_licenses_path} could not be found, installing it now...", file=sys.stderr)
            install_go_licenses()

    #package = get_package_name(args.go_mod_file)

    packages = get_dependencies(args.go_mod_file)
    print(f"Found {len(packages)} dependencies in {args.go_mod_file}")
    print(f"{packages}")
    for package in packages:
        print(f"Dependencies for {package}:")
        output = run_go_licenses(args.go_licenses_path, package)

        for line in output.splitlines():
            parts = line.split(",")
            if len(parts) != 3:
                print(f"Error: Unexpected format in go-licenses output: {line}", file=sys.stderr)
                sys.exit(2)
            dependency, _, license = parts
            print(f"{dependency}: {license}")


if __name__ == "__main__":
    main()
