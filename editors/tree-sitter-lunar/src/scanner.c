// The one token grammar.js cannot say: `await` is Lunar's keyword only when
// a name follows it on the same line (await f(), await self:m()), as the
// compiler reads it. Anywhere else it is Lua's name await: await(x),
// await (x), and `local a = await` then a new line.

#include "tree_sitter/parser.h"

enum TokenType {
  AWAIT,
};

void *tree_sitter_lunar_external_scanner_create(void) { return NULL; }

void tree_sitter_lunar_external_scanner_destroy(void *payload) { (void)payload; }

unsigned tree_sitter_lunar_external_scanner_serialize(void *payload, char *buffer) {
  (void)payload;
  (void)buffer;
  return 0;
}

void tree_sitter_lunar_external_scanner_deserialize(void *payload, const char *buffer, unsigned length) {
  (void)payload;
  (void)buffer;
  (void)length;
}

static bool is_name_start(int32_t c) { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'; }

static bool is_name_char(int32_t c) { return is_name_start(c) || (c >= '0' && c <= '9'); }

static bool is_space(int32_t c) { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'; }

bool tree_sitter_lunar_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid_symbols) {
  (void)payload;
  if (!valid_symbols[AWAIT]) {
    return false;
  }
  while (is_space(lexer->lookahead)) {
    lexer->advance(lexer, true);
  }
  for (const char *c = "await"; *c; c++) {
    if (lexer->lookahead != *c) {
      return false;
    }
    lexer->advance(lexer, false);
  }
  // awaiting is a name of its own.
  if (is_name_char(lexer->lookahead)) {
    return false;
  }
  lexer->mark_end(lexer);
  while (lexer->lookahead == ' ' || lexer->lookahead == '\t') {
    lexer->advance(lexer, false);
  }
  if (!is_name_start(lexer->lookahead)) {
    return false;
  }
  lexer->result_symbol = AWAIT;
  return true;
}
