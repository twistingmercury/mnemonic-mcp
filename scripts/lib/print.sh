# Print info message
print::info() {
    printf "[INFO] %s\n" "${1}"
}

# Print success message
print::success() {
    printf "[SUCCESS] %s\n" "${1}"
}

# Print error message to stderr
print::error() {
    printf "[ERROR] %s\n" "${1}" >&2
}

# Print warning message
print::warning() {
    printf "[WARNING] %s\n" "${1}"
}
