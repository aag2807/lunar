/*
 * nolibs: load a Lua chunk the way the engine loads a data file and print
 * the table it returns.
 *
 * The state is created with luaL_newstate() and no library is opened, so a
 * chunk that calls tostring, setmetatable, table.insert or anything else
 * global fails here with "attempt to call a nil value".
 *
 * Usage: nolibs <file.lua>
 * Exit status: 0 on success, 1 when the chunk fails or returns a non-table,
 * 2 on bad usage.
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <lua.h>
#include <lauxlib.h>

#define MAX_DEPTH 32

typedef struct {
    int is_number;
    lua_Number number;
    const char *string;
} Key;

/* Integer-like keys sort first, ascending; then string keys, bytewise. */
static int compare_keys(const void *a, const void *b) {
    const Key *x = a, *y = b;
    if (x->is_number != y->is_number) {
        return x->is_number ? -1 : 1;
    }
    if (x->is_number) {
        return (x->number > y->number) - (x->number < y->number);
    }
    return strcmp(x->string, y->string);
}

static void indent(int depth) {
    for (int i = 0; i < depth; i++) {
        fputs("  ", stdout);
    }
}

static void dump_table(lua_State *L, int index, int depth);

static void dump_value(lua_State *L, int index, int depth) {
    switch (lua_type(L, index)) {
    case LUA_TNIL:
        fputs("nil", stdout);
        break;
    case LUA_TBOOLEAN:
        fputs(lua_toboolean(L, index) ? "true" : "false", stdout);
        break;
    case LUA_TNUMBER:
        /* lua_tolstring converts in place; work on a copy of the slot. */
        lua_pushvalue(L, index);
        fputs(lua_tostring(L, -1), stdout);
        lua_pop(L, 1);
        break;
    case LUA_TSTRING:
        printf("\"%s\"", lua_tostring(L, index));
        break;
    case LUA_TTABLE:
        fputs("{\n", stdout);
        dump_table(L, index, depth + 1);
        indent(depth);
        fputs("}", stdout);
        break;
    default:
        printf("<%s>", luaL_typename(L, index));
        break;
    }
}

static void dump_table(lua_State *L, int index, int depth) {
    if (depth > MAX_DEPTH) {
        indent(depth);
        fputs("...\n", stdout);
        return;
    }
    index = lua_absindex(L, index);

    size_t count = 0, capacity = 16;
    Key *keys = malloc(capacity * sizeof *keys);
    if (keys == NULL) {
        fputs("nolibs: out of memory\n", stderr);
        exit(1);
    }

    lua_pushnil(L);
    while (lua_next(L, index) != 0) {
        lua_pop(L, 1); /* drop the value, keep the key */
        if (count == capacity) {
            capacity *= 2;
            Key *grown = realloc(keys, capacity * sizeof *keys);
            if (grown == NULL) {
                fputs("nolibs: out of memory\n", stderr);
                exit(1);
            }
            keys = grown;
        }
        Key *k = &keys[count++];
        if (lua_type(L, -1) == LUA_TNUMBER) {
            k->is_number = 1;
            k->number = lua_tonumber(L, -1);
            k->string = NULL;
        } else if (lua_type(L, -1) == LUA_TSTRING) {
            k->is_number = 0;
            k->number = 0;
            k->string = lua_tostring(L, -1); /* the table keeps the string alive */
        } else {
            count--; /* booleans, tables as keys: not expected in data files */
        }
    }

    qsort(keys, count, sizeof *keys, compare_keys);

    for (size_t i = 0; i < count; i++) {
        indent(depth);
        if (keys[i].is_number) {
            lua_pushnumber(L, keys[i].number);
            lua_Integer as_int = (lua_Integer)keys[i].number;
            if ((lua_Number)as_int == keys[i].number) {
                lua_pop(L, 1);
                lua_pushinteger(L, as_int);
                printf("[%lld] = ", (long long)as_int);
            } else {
                printf("[%s] = ", lua_tostring(L, -1));
            }
        } else {
            lua_pushstring(L, keys[i].string);
            printf("%s = ", keys[i].string);
        }
        lua_rawget(L, index);
        dump_value(L, -1, depth);
        lua_pop(L, 1);
        fputc('\n', stdout);
    }

    free(keys);
}

int main(int argc, char **argv) {
    if (argc != 2) {
        fputs("usage: nolibs <file.lua>\n", stderr);
        return 2;
    }

    lua_State *L = luaL_newstate(); /* deliberately no luaL_openlibs */
    if (L == NULL) {
        fputs("nolibs: cannot create Lua state\n", stderr);
        return 1;
    }

    if (luaL_loadfile(L, argv[1]) != LUA_OK || lua_pcall(L, 0, 1, 0) != LUA_OK) {
        fprintf(stderr, "nolibs: %s\n", lua_tostring(L, -1));
        lua_close(L);
        return 1;
    }

    if (!lua_istable(L, -1)) {
        fprintf(stderr, "nolibs: %s must return a table, got %s\n", argv[1], luaL_typename(L, -1));
        lua_close(L);
        return 1;
    }

    dump_table(L, -1, 0);
    lua_close(L);
    return 0;
}
