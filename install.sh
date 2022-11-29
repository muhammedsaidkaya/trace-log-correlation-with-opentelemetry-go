
# Brew Installation
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Kubectl Installation
brew install kubectl

# Minikube Installation
brew install minikube
minikube start --profile custom

# Skaffold Installation
brew install skaffold
skaffold config set --global collect-metrics false
skaffold config set --global local-cluster true