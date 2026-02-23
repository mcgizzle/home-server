syntax on

filetype plugin indent on

set number
set showmode
set smartcase
set smarttab
set smartindent
set autoindent
set expandtab
set shiftwidth=2
set softtabstop=2

if $VIM_CRONTAB == "true"
  set nobackup
  set nowritebackup
endif
execute pathogen#infect()

call neomake#configure#automake('w')

let g:neomake_javascript_enabled_makers = ['eslint']
let g:deoplete#enable_at_startup = 1
let g:scala_scaladoc_indent = 1
let g:elm_format_autosave = 0
" let g:deoplete#disable_auto_complete = 1

autocmd BufWritePre *.js Neoformat

" NerdTree Ctrl-e
map <C-e> :NERDTreeToggle<CR>

map ff <S-g>

let g:hindent_on_save = 1
let g:deoplete#enable_at_startup = 1
let NERDTreeShowHidden=1

"autocmd VimEnter * colorscheme evening
au FileType xml exe ":silent %!xmllint --format --recover - 2>/dev/null"

