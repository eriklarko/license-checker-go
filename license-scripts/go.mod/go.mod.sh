#! /usr/bin/env bash

# This script is designed to output the licenses for all the dependencies to the
# console.
#
# Example:
#  Given the following directory structure:
#    .
#    ├── go.mod
#    ├── go.mod.sh
#    └── main.go
#
#   Simply run the script from the root:
#     > ./go.mod.sh
#     github.com/foo/bar: MIT
#     github.com/baz/qux: Apache-2.0
#     golang.org/x/crypto: BSD-3-Clause
#     gopkg.in/yaml.v3: MIT
#
#
# This script uses the go-licenses tool from https://github.com/google/go-licenses.
# It needs to be installed before this script can be used, but this script handles
# that for you.
#
#
# NOTE: This script is provided "as is", without warranty of any kind, express or implied.
# The authors and contributors of this script take no responsibility for any consequences
# resulting from the use of this script. Users are advised to review and understand the
# script's functionality before use and to use it at their own risk. It is the user's
# responsibility to ensure compliance with all applicable licenses and legal requirements.
#
#
# To specify a different path for the go-licenses tool or go.mod file, specify
# the path as an argument to the script.
#
# Example:
#   ./go.mod.sh --go-licenses-path /path/to/my-go-licenses-tool --go-mod-file /path/to/my-go.mod
#
# You can also set the GO_LICENSES_PATH and GO_MOD_FILE environment variables
# before running the script, but note that command line arguments take
# precedence.
#
# Example:
#   GO_LICENSES_PATH=/path/to/my-go-licenses-tool ./go.mod.sh
#   GO_MOD_FILE=/path/to/my-go.mod go ./go.mod.sh
#   
#  and both at the same time for nice completeness <3 and it's not obvious how
#  to separate env vars on the command line
#   GO_LICENSES_PATH=/path/to/my-go-licenses-tool GO_MOD_FILE=/path/to/my-go.mod ./go.mod.sh


# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --go-licenses-path)
            GO_LICENSES_PATH="$2"
            shift 2
            ;;
        --go-mod-file)
            GO_MOD_FILE="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1
            ;;
    esac
done

# Set defaults if not provided by command line arguments or environment variables
GO_LICENSES_PATH="${GO_LICENSES_PATH:-go-licenses}"
GO_MOD_FILE="${GO_MOD_FILE:-go.mod}"

# check that the go.mod file exists
if [ ! -f "$GO_MOD_FILE" ]; then
    # Make GO_MOD_FILE path absolute so it's easier to find if things go wrong.
    # if we printed just "go.mod file does not exist" it would be hard to know
    # where we're looking for it, especially in CI environments. So printing the
    # absolute path makes debugging a little easier, love ya next dev.
    GO_MOD_FILE_ABS=$(realpath "$GO_MOD_FILE" 2>/dev/null) || $GO_MOD_FILE

    echo "Error: $GO_MOD_FILE_ABS file not found" >&2
    exit 1
fi

# if the go-licenses tool is not installed, install it
if ! command -v "$GO_LICENSES_PATH" &> /dev/null
then
    echo "${GO_LICENSES_PATH} could not be found, installing it now..."
    go install github.com/google/go-licenses@latest
fi

# The go-licenses tool requires the package name to be passed as an argument, so
# we first read it from the go.mod file

## Read the first line of the go.mod file and extract the package name
## The first line is expected to be in the format: module <package-name>
## Example: module github.com/some-org/some-repo
PACKAGE=$(head -n 1 "$GO_MOD_FILE" | awk '{print $2}')

# Run go-licenses report with the extracted package name
# The output of the go-licenses tool is in format DEPENDENCY,URL,LICENSE and we
# just want the dependency and license.
$GO_LICENSES_PATH report "$PACKAGE" | while IFS=',' read -r dependency _ license; do
    echo "$dependency: $license"
done
