##################################################################
# DOT Files                                                      #
##################################################################

alias dotcfg='git -C $HOME/code/personal/home-server'
alias zshcfg='nvim ~/.zshrc'
alias aliascfg='nvim ~/.aliases.sh'
alias envcfg='nvim ~/.zshenv'
alias reload='exec zsh && source ~/.zshenv'

alias gp='gcloud config configurations activate personal'
alias gw='gcloud config configurations activate work'

##################################################################
# General                                                        #
##################################################################
alias dir='basename `PWD`'

alias v="nvim"

alias cd_root='cd $(git rev-parse --show-toplevel)'

alias tf="terraform"

function :w () {
  exit
}
function :q () {
  exit
}
function :wq () {
  exit
}

trash () {
  mv "$1" "$HOME/.Trash"
}

ignore (){
  echo "$1" >> .gitignore
}

##################################################################
# Git                                                      #
##################################################################

gr="gcm && gfo && groh"

## Taken from: https://github.com/ohmyzsh/ohmyzsh/blob/master/plugins/git/git.plugin.zsh

# Check if main exists and use instead of master
function git_main_branch() {
   command git rev-parse --git-dir &>/dev/null || return
     local ref
       for ref in refs/{heads,remotes/{origin,upstream}}/{main,trunk,mainline,default,stable,master}; do
           if command git show-ref -q --verify $ref; then
                 echo ${ref:t}
                       return 0
                           fi
                             done

                               # If no main branch was found, fall back to master but return error
                                 echo master
                                   return 1
                                 }

function git_current_branch() {
  git rev-parse --abbrev-ref HEAD
}

alias g='git'
alias gc='git commit --verbose'
alias ga='g add'
alias ggpush='git push origin "$(git_current_branch)"'
alias gfpush='ggpush -f'
alias ggpull='git pull origin "$(git_current_branch)"'
alias gcma='git checkout $(git_main_branch)'
alias gfo='git fetch origin'
alias gpu='git push upstream'
alias gpristine='git reset --hard && git clean --force -dfx'
alias gwipe='git reset --hard && git clean --force -df'
alias groh='git reset origin/$(git_current_branch) --hard'
alias gsh='git show'
alias gsps='git show --pretty=short --show-signature'
alias gsts='git stash show --patch'
alias gst='git status'
alias gss='git status --short'
alias gsb='git status --short --branch'
alias gco='git checkout'
alias gcb='git checkout -b'
alias gcB='git checkout -B'
alias gcd='git checkout $(git_develop_branch)'
alias gcm='git checkout $(git_main_branch)'
alias gcl='git clone'
alias gclean='git clean --interactive -d'
alias gca='git commit --verbose --all'
alias gd='git diff'

##################################################################
# Vim                                                            #
##################################################################

alias vim='nvim'
alias pathonv='cd ~/.config/nvim/bundle'
alias vimcfg='nvim ~/.config/nvim/init.vim'

##################################################################
# Kubernetes                                                     #
##################################################################

alias kctx='kubectl config use-context '"$1"''

##################################################################
# Docker                                                         #
##################################################################

function docker-upf() {
  docker-compose up -d "$1" && docker-compose logs -f "$1"
}

function docker-stop() {
  docker stop $(docker ps | grep "$1" | cut -f 1 -d " ")
}
