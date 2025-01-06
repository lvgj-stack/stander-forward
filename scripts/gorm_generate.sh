PROJECT_DIR=$(dirname $(dirname "$0"))
GENERATE_DIR="$PROJECT_DIR/dal/cmd"

cd "$GENERATE_DIR" || exit

echo "Start Generating $GENERATE_DIR/generate.go"
go run .