; Zed indentation: a node indents its contents up to its closing token.
(_ "end" @end) @indent
(_ "until" @end) @indent
(_ "{" "}" @end) @indent
(_ "(" ")" @end) @indent
