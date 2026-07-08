#!/bin/bash

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRET_FILE="$SCRIPT_DIR/confluence.secret"

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

detect_platform() {
    local os_name=$(uname -s)
    local os_arch=$(uname -m)
    
    case "$os_name" in
        Darwin)
            PLATFORM="darwin"
            ;;
        Linux)
            PLATFORM="linux"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            PLATFORM="windows"
            ;;
        *)
            print_error "Unsupported operating system: $os_name"
            exit 1
            ;;
    esac
    
    case "$os_arch" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        i386|i686)
            ARCH="amd64"
            ;;
        *)
            print_error "Unsupported architecture: $os_arch"
            exit 1
            ;;
    esac
    
    PLATFORM_DIR="${PLATFORM}-${ARCH}"
    
    if [[ "$PLATFORM" == "windows" ]]; then
        BINARY_NAME="markdown2confluence.exe"
    else
        BINARY_NAME="markdown2confluence"
    fi
    
    BINARY_PATH="$SCRIPT_DIR/bin/$PLATFORM_DIR/$BINARY_NAME"
}

check_binary() {
    if [[ ! -f "$BINARY_PATH" ]]; then
        print_error "Binary not found at: $BINARY_PATH"
        print_info "Please build the binary first or check your installation"
        exit 1
    fi
    
    if [[ "$PLATFORM" != "windows" ]]; then
        chmod +x "$BINARY_PATH"
    fi
    
    print_success "Found binary: $BINARY_PATH"
}

add_to_path() {
    local bin_dir="$SCRIPT_DIR/bin/$PLATFORM_DIR"
    local config_file=""
    
    if [[ "$PLATFORM" == "windows" ]]; then
        print_warning "Windows detected. Please manually add the following directory to your PATH:"
        echo "  $bin_dir"
        return
    fi
    
    if [[ -n "$ZSH_VERSION" ]]; then
        config_file="$HOME/.zshrc"
    elif [[ -n "$BASH_VERSION" ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            config_file="$HOME/.bashrc"
        elif [[ -f "$HOME/.bash_profile" ]]; then
            config_file="$HOME/.bash_profile"
        fi
    else
        config_file="$HOME/.profile"
    fi
    
    if [[ -z "$config_file" ]]; then
        config_file="$HOME/.bashrc"
    fi
    
    if [[ -f "$config_file" ]]; then
        if grep -q "markdown2confluence" "$config_file" 2>/dev/null; then
            print_info "PATH already configured in $config_file"
        else
            echo "" >> "$config_file"
            echo "# Added by markdown2confluence installer" >> "$config_file"
            echo "export PATH=\"\$PATH:$bin_dir\"" >> "$config_file"
            print_success "Added $bin_dir to PATH in $config_file"
            print_info "Run 'source $config_file' or start a new terminal to use the command"
        fi
    fi
}

create_secret_file() {
    print_info "Setting up Confluence credentials..."
    echo ""
    echo -e "${YELLOW}You need to create a Personal Access Token at:${NC}"
    echo -e "${BLUE}https://wiki.amh-group.com/plugins/tokens/view.action${NC}"
    echo ""
    
    if [[ -f "$SECRET_FILE" ]]; then
        print_warning "Found existing $SECRET_FILE"
        read -p "Do you want to overwrite it? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Keeping existing configuration"
            return
        fi
    fi
    
    read -p "Enter your Confluence username (email): " username
    if [[ -z "$username" ]]; then
        print_error "Username cannot be empty"
        exit 1
    fi
    
    read -p "Enter your Confluence Personal Access Token: " -s token
    echo
    if [[ -z "$token" ]]; then
        print_error "Token cannot be empty"
        exit 1
    fi
    
    cat > "$SECRET_FILE" << EOF
export CONFLUENCE_USERNAME=$username
export CONFLUENCE_PASSWORD=
export CONFLUENCE_ENDPOINT=https://wiki.amh-group.com
export CONFLUENCE_PERSONAL_ACCESS_TOKEN=$token
EOF
    
    chmod 600 "$SECRET_FILE"
    print_success "Created $SECRET_FILE"
    print_info "You can source this file in your shell: source $SECRET_FILE"
}

main() {
    echo "========================================="
    echo "  markdown2confluence Initializer"
    echo "========================================="
    echo ""
    
    print_info "Detecting platform..."
    detect_platform
    print_success "Detected: $PLATFORM_DIR"
    
    print_info "Checking binary..."
    check_binary
    
    print_info "Adding to PATH..."
    add_to_path
    
    create_secret_file
    
    echo ""
    echo "========================================="
    print_success "Installation complete!"
    echo "========================================="
    echo ""
    print_info "Next steps:"
    echo "  1. Start a new terminal or run: source ~/.bashrc (or ~/.zshrc)"
    echo "  2. Load credentials: source $SECRET_FILE"
    echo "  3. Run: markdown2confluence --help"
    echo ""
}

main "$@"