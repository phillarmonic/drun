#!/bin/bash
set -euo pipefail

# drun installer script
# Usage: curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
# Usage: curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s v1.0.0

# Configuration
REPO="phillarmonic/drun"
BINARY_NAME="drun"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
GITHUB_API="https://api.github.com/repos/${REPO}"
GITHUB_RELEASES="https://github.com/${REPO}/releases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

# Cleanup function
cleanup() {
    if [[ -n "${TEMP_DIR:-}" ]] && [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Set up cleanup trap
trap cleanup EXIT

# Check if running on supported platform
check_platform() {
    local os arch
    
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            os="windows"
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            log_error "Supported platforms: Linux, macOS, Windows"
            exit 1
            ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            log_error "Supported architectures: amd64, arm64"
            exit 1
            ;;
    esac
    
    PLATFORM_OS="$os"
    PLATFORM_ARCH="$arch"
    
    # Set binary name with extension for Windows
    if [[ "$os" == "windows" ]]; then
        RELEASE_BINARY="drun-${os}-${arch}.exe"
    else
        RELEASE_BINARY="drun-${os}-${arch}"
    fi
    
    log_info "Detected platform: ${PLATFORM_OS}/${PLATFORM_ARCH}"
}

# Check if required tools are available
check_dependencies() {
    local missing_deps=()
    
    for cmd in curl tar; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_deps+=("$cmd")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install the missing tools and try again"
        exit 1
    fi
}

# Get the latest release version from GitHub API
get_latest_version() {
    log_info "Fetching latest release information..." >&2
    
    local latest_url="${GITHUB_API}/releases/latest"
    local response
    
    if ! response=$(curl -sSf "$latest_url" 2>/dev/null); then
        log_error "Failed to fetch release information from GitHub"
        log_error "Please check your internet connection or try again later"
        exit 1
    fi
    
    # Extract version tag from JSON response
    local version
    if command -v jq >/dev/null 2>&1; then
        version=$(echo "$response" | jq -r '.tag_name')
    else
        # Fallback parsing without jq
        version=$(echo "$response" | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [[ -z "$version" || "$version" == "null" ]]; then
        log_error "Failed to parse latest version from GitHub API"
        exit 1
    fi
    
    echo "$version"
}

# Validate version format
validate_version() {
    local version="$1"
    
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        log_error "Invalid version format: $version"
        log_error "Expected format: v1.0.0 or v1.0.0-beta.1"
        exit 1
    fi
}

# Check if version exists
check_version_exists() {
    local version="$1"
    local releases_url="${GITHUB_API}/releases/tags/${version}"
    
    log_info "Checking if version $version exists..."
    
    if ! curl -sSf "$releases_url" >/dev/null 2>&1; then
        log_error "Version $version not found"
        log_error "Available releases: ${GITHUB_RELEASES}"
        exit 1
    fi
    
    log_success "Version $version found"
}

# Download and install binary
install_binary() {
    local version="$1"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${RELEASE_BINARY}"
    
    log_info "Creating temporary directory..."
    TEMP_DIR=$(mktemp -d)
    
    local temp_binary="${TEMP_DIR}/${BINARY_NAME}"
    
    log_info "Downloading ${RELEASE_BINARY}..."
    log_info "URL: $download_url"
    
    if ! curl -sSfL "$download_url" -o "$temp_binary"; then
        log_error "Failed to download binary from $download_url"
        log_error "Please check if the release exists: ${GITHUB_RELEASES}/tag/${version}"
        exit 1
    fi
    
    # Make binary executable
    chmod +x "$temp_binary"
    
    # Verify the binary works
    log_info "Verifying binary..."
    if ! "$temp_binary" --version >/dev/null 2>&1; then
        log_error "Downloaded binary failed verification"
        exit 1
    fi
    
    # Check if install directory exists and is writable
    if [[ ! -d "$INSTALL_DIR" ]]; then
        log_error "Install directory does not exist: $INSTALL_DIR"
        log_error "Please create it or set INSTALL_DIR environment variable"
        exit 1
    fi
    
    if [[ ! -w "$INSTALL_DIR" ]]; then
        log_warn "Install directory is not writable: $INSTALL_DIR"
        log_info "Attempting to install with sudo..."
        
        if ! sudo mv "$temp_binary" "${INSTALL_DIR}/${BINARY_NAME}"; then
            log_error "Failed to install binary to $INSTALL_DIR"
            log_error "Please check permissions or try a different install directory"
            exit 1
        fi
    else
        if ! mv "$temp_binary" "${INSTALL_DIR}/${BINARY_NAME}"; then
            log_error "Failed to install binary to $INSTALL_DIR"
            exit 1
        fi
    fi
    
    log_success "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Verify installation
verify_installation() {
    local installed_binary="${INSTALL_DIR}/${BINARY_NAME}"
    
    if [[ ! -f "$installed_binary" ]]; then
        log_error "Binary not found at $installed_binary"
        exit 1
    fi
    
    if [[ ! -x "$installed_binary" ]]; then
        log_error "Binary is not executable: $installed_binary"
        exit 1
    fi
    
    log_info "Verifying installation..."
    local version_output
    if ! version_output=$("$installed_binary" --version 2>&1); then
        log_error "Failed to run installed binary"
        log_error "Output: $version_output"
        exit 1
    fi
    
    log_success "Installation verified successfully!"
    log_info "Version: $version_output"
}

# Check if binary is in PATH
check_path() {
    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        log_warn "$INSTALL_DIR is not in your PATH"
        log_info "Add the following to your shell profile (.bashrc, .zshrc, etc.):"
        log_info "export PATH=\"\$PATH:$INSTALL_DIR\""
        log_info ""
        log_info "Or run: echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.bashrc"
        log_info "Then restart your shell or run: source ~/.bashrc"
    else
        log_success "$INSTALL_DIR is in your PATH"
    fi
}

# Show usage instructions
show_usage() {
    cat << EOF
drun installer

USAGE:
    curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
    curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s [VERSION]

EXAMPLES:
    # Install latest version
    curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash
    
    # Install specific version
    curl -sSL https://raw.githubusercontent.com/phillarmonic/drun/master/install.sh | bash -s v1.0.0

ENVIRONMENT VARIABLES:
    INSTALL_DIR    Installation directory (default: /usr/local/bin)

REQUIREMENTS:
    - curl
    - tar
    - Linux, macOS, or Windows
    - amd64 or arm64 architecture

EOF
}

# Main installation function
main() {
    local version="${1:-}"
    
    # Show help if requested
    if [[ "$version" == "-h" || "$version" == "--help" ]]; then
        show_usage
        exit 0
    fi
    
    echo "🚀 drun installer"
    echo "=================="
    echo ""
    
    # Check platform and dependencies
    check_platform
    check_dependencies
    
    # Determine version to install
    if [[ -z "$version" ]]; then
        version=$(get_latest_version)
        log_info "Installing latest version: $version"
    else
        validate_version "$version"
        check_version_exists "$version"
        log_info "Installing specified version: $version"
    fi
    
    # Install the binary
    install_binary "$version"
    
    # Verify installation
    verify_installation
    
    # Check PATH
    check_path
    
    echo ""
    log_success "🎉 drun installation completed successfully!"
    echo ""
    log_info "Get started with:"
    log_info "  drun --help"
    log_info "  drun --init"
    log_info ""
    log_info "Documentation: https://github.com/${REPO}"
    log_info "Examples: https://github.com/${REPO}/tree/master/examples"
}

# Run main function with all arguments
main "$@"
