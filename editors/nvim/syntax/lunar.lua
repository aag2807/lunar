-- Lunar syntax highlighting for Neovim
-- Uses Lua API for syntax highlighting

if vim.b.current_syntax then
  return
end

local ts_highlight = vim.treesitter.highlighter.active[vim.api.nvim_get_current_buf()]
if ts_highlight then
  return
end

-- Keywords
vim.cmd([[
  syntax keyword lunarKeyword function end return local if then else elseif while do for in repeat until break
  syntax keyword lunarKeyword class constructor private public protected static extends implements interface
  syntax keyword lunarKeyword async await export import from as type declare namespace module
  syntax keyword lunarKeyword const let var enum struct trait

  syntax keyword lunarBoolean true false
  syntax keyword lunarConstant nil
  syntax keyword lunarOperator and or not

  " Types
  syntax keyword lunarType number string boolean any void table function never unknown
  syntax keyword lunarType u8 u16 u32 u64 i8 i16 i32 i64 f32 f64

  " Built-in functions
  syntax keyword lunarBuiltin print type tonumber tostring pairs ipairs next
  syntax keyword lunarBuiltin setmetatable getmetatable rawget rawset rawequal
  syntax keyword lunarBuiltin require pcall xpcall error assert
  syntax keyword lunarBuiltin coroutine string table math io os debug
  syntax keyword lunarSpecial self

  " Comments
  syntax match lunarComment "--.*$"
  syntax region lunarBlockComment start="--\[\[" end="\]\]"

  " Strings
  syntax region lunarString start=+"+ skip=+\\\\\|\\\"+ end=+"+
  syntax region lunarString start=+'+ skip=+\\\\\|\\\'+ end=+'+
  syntax region lunarString start=+\[\[+ end=+\]\]+

  " Numbers
  syntax match lunarNumber "\<\d\+\>"
  syntax match lunarNumber "\<\d\+\.\d*\>"
  syntax match lunarNumber "\.\d\+\>"
  syntax match lunarNumber "\<\d\+[eE][+-]\=\d\+\>"
  syntax match lunarNumber "0[xX]\x\+\>"
  syntax match lunarNumber "0[bB][01]\+\>"

  " Type annotations
  syntax match lunarTypeAnnotation ":\s*\w\+\>" contains=lunarType
  syntax match lunarTypeAnnotation "<\w\+>" contains=lunarType

  " Operators
  syntax match lunarOperator "+"
  syntax match lunarOperator "-"
  syntax match lunarOperator "\*"
  syntax match lunarOperator "/"
  syntax match lunarOperator "%"
  syntax match lunarOperator "\^"
  syntax match lunarOperator "\.\."
  syntax match lunarOperator "=="
  syntax match lunarOperator "\~="
  syntax match lunarOperator "<="
  syntax match lunarOperator ">="
  syntax match lunarOperator "<"
  syntax match lunarOperator ">"
  syntax match lunarOperator "="
  syntax match lunarOperator "??"
  syntax match lunarOperator "?."
  syntax match lunarOperator "|>"
  syntax match lunarOperator "=>"
  syntax match lunarOperator "->"

  " Functions
  syntax match lunarFunction "\<\w\+\ze\s*("

  " Decorators
  syntax match lunarDecorator "@\w\+"

  " Special characters
  syntax match lunarDelimiter "[(){}\[\],;]"
]])

-- Link to standard highlight groups
vim.cmd([[
  highlight default link lunarKeyword Keyword
  highlight default link lunarBoolean Boolean
  highlight default link lunarConstant Constant
  highlight default link lunarOperator Operator
  highlight default link lunarType Type
  highlight default link lunarBuiltin Function
  highlight default link lunarSpecial Special
  highlight default link lunarComment Comment
  highlight default link lunarBlockComment Comment
  highlight default link lunarString String
  highlight default link lunarNumber Number
  highlight default link lunarTypeAnnotation Type
  highlight default link lunarFunction Function
  highlight default link lunarDecorator PreProc
  highlight default link lunarDelimiter Delimiter
]])

vim.b.current_syntax = "lunar"
