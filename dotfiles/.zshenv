export VISUAL="nvim"
export EDITOR="nvim"
export SBT_CREDENTIALS=${HOME}/.sbt/.credentials
export CLOUDSDK_PYTHON=/usr/bin/python3
export PATH=${PATH}:$HOME/.local/bin
export PATH=${PATH}:$HOME/code/permutive/blobby/bin
export PATH=${PATH}:$HOME/code/scripts/db-proxy/bin
export PATH=${PATH}:$HOME/code/scripts/kafka/bin
export PATH=${PATH}:$HOME/Library/Python/3.8/bin
export PATH=${PATH}:$HOME/code/graalvm-jdk-17.0.7+8.1/Contents/Home/bin
export PATH=${PATH}:/opt/homebrew/bin
export PATH=${PATH}:$HOME/binaries/globby-0.1.0-Darwin-arm64
export PATH="$HOME/Library/Python/3.9/bin:$PATH"
export PATH=$PATH:$(go env GOPATH)/bin
export PATH="/opt/homebrew/opt/postgresql@17/bin:$PATH"

# AWS Bedrock (used by Claude Code + OpenClaw)
export AWS_PROFILE="bedrock-access-316332150940"
export AWS_REGION="us-east-1"

