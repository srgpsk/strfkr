#!/bin/bash
# Install git hooks

echo "Installing Git hooks..."

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "⚠️  Not in a git repository. Git hooks will be configured when repo is initialized."
else
    # Make hooks executable
    chmod +x .githooks/pre-commit
    chmod +x .githooks/pre-push
    chmod +x .githooks/commit-msg
    
    # Install hooks
    git config core.hooksPath .githooks
    
    # Configure GPG signing (if GPG key is available)
    if gpg --list-secret-keys --keyid-format=long | grep -q "sec"; then
        echo "🔐 Configuring commit signing..."
        git config commit.gpgsign true
        git config tag.gpgsign true
        echo "✅ Commit signing enabled!"
    else
        echo "⚠️  No GPG key found. Commit signing not configured."
    fi
    
    echo "✅ Git hooks installed!"
fi

echo ""
echo "Hooks configured:"
echo "  📝 pre-commit: Format, lint, vet (no tests)"
echo "  🧪 pre-push: Full test suite with coverage"
echo "  📄 commit-msg: Enforce conventional commit format"
echo "  🔐 commit-signing: $([ -n "$(gpg --list-secret-keys)" ] && echo "Enabled" || echo "Disabled (no keys)")"