-- Lua the compiler used to change: bracketed name keys, parenthesised
-- callees and index bases, ';' before '(', a '(' starting a statement,
-- long elseif chains, and ':' calls on vararg functions in tables.
local KEY = "hp"
local t = { [KEY] = 100, plain = 1 }
print(t.hp, t.KEY, t.plain)

local g = function(x) return x * 10 end
local holder = {}
print((holder.f or g)(2))
local a, b = nil, { "first" }
print((a or b)[1])
local r = (function(x) return x + 1 end)(4)
print(r)
local function three() return 1, 2, 3 end
print(({three()})[3])

local f = print
;("x"):rep(2)
f("after semicolon")

local u = {}
;(function() print("own statement") end)()
local v = {}
(function() print("own statement 2") end)()

local o = { log = function(self, ...) return select("#", ...) end }
print(o:log(1, 2))

local function pick(n)
    if n == 1 then return 1
elseif n == 2 then return 2
elseif n == 3 then return 3
elseif n == 4 then return 4
elseif n == 5 then return 5
elseif n == 6 then return 6
elseif n == 7 then return 7
elseif n == 8 then return 8
elseif n == 9 then return 9
elseif n == 10 then return 10
elseif n == 11 then return 11
elseif n == 12 then return 12
elseif n == 13 then return 13
elseif n == 14 then return 14
elseif n == 15 then return 15
elseif n == 16 then return 16
elseif n == 17 then return 17
elseif n == 18 then return 18
elseif n == 19 then return 19
elseif n == 20 then return 20
elseif n == 21 then return 21
elseif n == 22 then return 22
elseif n == 23 then return 23
elseif n == 24 then return 24
elseif n == 25 then return 25
elseif n == 26 then return 26
elseif n == 27 then return 27
elseif n == 28 then return 28
elseif n == 29 then return 29
elseif n == 30 then return 30
elseif n == 31 then return 31
elseif n == 32 then return 32
elseif n == 33 then return 33
elseif n == 34 then return 34
elseif n == 35 then return 35
elseif n == 36 then return 36
elseif n == 37 then return 37
elseif n == 38 then return 38
elseif n == 39 then return 39
elseif n == 40 then return 40
elseif n == 41 then return 41
elseif n == 42 then return 42
elseif n == 43 then return 43
elseif n == 44 then return 44
elseif n == 45 then return 45
elseif n == 46 then return 46
elseif n == 47 then return 47
elseif n == 48 then return 48
elseif n == 49 then return 49
elseif n == 50 then return 50
elseif n == 51 then return 51
elseif n == 52 then return 52
elseif n == 53 then return 53
elseif n == 54 then return 54
elseif n == 55 then return 55
elseif n == 56 then return 56
elseif n == 57 then return 57
elseif n == 58 then return 58
elseif n == 59 then return 59
elseif n == 60 then return 60
elseif n == 61 then return 61
elseif n == 62 then return 62
elseif n == 63 then return 63
elseif n == 64 then return 64
elseif n == 65 then return 65
elseif n == 66 then return 66
elseif n == 67 then return 67
elseif n == 68 then return 68
elseif n == 69 then return 69
elseif n == 70 then return 70
elseif n == 71 then return 71
elseif n == 72 then return 72
elseif n == 73 then return 73
elseif n == 74 then return 74
elseif n == 75 then return 75
elseif n == 76 then return 76
elseif n == 77 then return 77
elseif n == 78 then return 78
elseif n == 79 then return 79
elseif n == 80 then return 80
elseif n == 81 then return 81
elseif n == 82 then return 82
elseif n == 83 then return 83
elseif n == 84 then return 84
elseif n == 85 then return 85
elseif n == 86 then return 86
elseif n == 87 then return 87
elseif n == 88 then return 88
elseif n == 89 then return 89
elseif n == 90 then return 90
elseif n == 91 then return 91
elseif n == 92 then return 92
elseif n == 93 then return 93
elseif n == 94 then return 94
elseif n == 95 then return 95
elseif n == 96 then return 96
elseif n == 97 then return 97
elseif n == 98 then return 98
elseif n == 99 then return 99
elseif n == 100 then return 100
elseif n == 101 then return 101
elseif n == 102 then return 102
elseif n == 103 then return 103
elseif n == 104 then return 104
elseif n == 105 then return 105
elseif n == 106 then return 106
elseif n == 107 then return 107
elseif n == 108 then return 108
elseif n == 109 then return 109
elseif n == 110 then return 110
elseif n == 111 then return 111
elseif n == 112 then return 112
elseif n == 113 then return 113
elseif n == 114 then return 114
elseif n == 115 then return 115
elseif n == 116 then return 116
elseif n == 117 then return 117
elseif n == 118 then return 118
elseif n == 119 then return 119
elseif n == 120 then return 120
elseif n == 121 then return 121
elseif n == 122 then return 122
elseif n == 123 then return 123
elseif n == 124 then return 124
elseif n == 125 then return 125
elseif n == 126 then return 126
elseif n == 127 then return 127
elseif n == 128 then return 128
elseif n == 129 then return 129
elseif n == 130 then return 130
elseif n == 131 then return 131
elseif n == 132 then return 132
elseif n == 133 then return 133
elseif n == 134 then return 134
elseif n == 135 then return 135
elseif n == 136 then return 136
elseif n == 137 then return 137
elseif n == 138 then return 138
elseif n == 139 then return 139
elseif n == 140 then return 140
elseif n == 141 then return 141
elseif n == 142 then return 142
elseif n == 143 then return 143
elseif n == 144 then return 144
elseif n == 145 then return 145
elseif n == 146 then return 146
elseif n == 147 then return 147
elseif n == 148 then return 148
elseif n == 149 then return 149
elseif n == 150 then return 150
elseif n == 151 then return 151
elseif n == 152 then return 152
elseif n == 153 then return 153
elseif n == 154 then return 154
elseif n == 155 then return 155
elseif n == 156 then return 156
elseif n == 157 then return 157
elseif n == 158 then return 158
elseif n == 159 then return 159
elseif n == 160 then return 160
elseif n == 161 then return 161
elseif n == 162 then return 162
elseif n == 163 then return 163
elseif n == 164 then return 164
elseif n == 165 then return 165
elseif n == 166 then return 166
elseif n == 167 then return 167
elseif n == 168 then return 168
elseif n == 169 then return 169
elseif n == 170 then return 170
elseif n == 171 then return 171
elseif n == 172 then return 172
elseif n == 173 then return 173
elseif n == 174 then return 174
elseif n == 175 then return 175
elseif n == 176 then return 176
elseif n == 177 then return 177
elseif n == 178 then return 178
elseif n == 179 then return 179
elseif n == 180 then return 180
elseif n == 181 then return 181
elseif n == 182 then return 182
elseif n == 183 then return 183
elseif n == 184 then return 184
elseif n == 185 then return 185
elseif n == 186 then return 186
elseif n == 187 then return 187
elseif n == 188 then return 188
elseif n == 189 then return 189
elseif n == 190 then return 190
elseif n == 191 then return 191
elseif n == 192 then return 192
elseif n == 193 then return 193
elseif n == 194 then return 194
elseif n == 195 then return 195
elseif n == 196 then return 196
elseif n == 197 then return 197
elseif n == 198 then return 198
elseif n == 199 then return 199
elseif n == 200 then return 200
elseif n == 201 then return 201
elseif n == 202 then return 202
elseif n == 203 then return 203
elseif n == 204 then return 204
elseif n == 205 then return 205
elseif n == 206 then return 206
elseif n == 207 then return 207
elseif n == 208 then return 208
elseif n == 209 then return 209
elseif n == 210 then return 210
elseif n == 211 then return 211
elseif n == 212 then return 212
elseif n == 213 then return 213
elseif n == 214 then return 214
elseif n == 215 then return 215
elseif n == 216 then return 216
elseif n == 217 then return 217
elseif n == 218 then return 218
elseif n == 219 then return 219
elseif n == 220 then return 220
elseif n == 221 then return 221
elseif n == 222 then return 222
elseif n == 223 then return 223
elseif n == 224 then return 224
elseif n == 225 then return 225
elseif n == 226 then return 226
elseif n == 227 then return 227
elseif n == 228 then return 228
elseif n == 229 then return 229
elseif n == 230 then return 230
elseif n == 231 then return 231
elseif n == 232 then return 232
elseif n == 233 then return 233
elseif n == 234 then return 234
elseif n == 235 then return 235
elseif n == 236 then return 236
elseif n == 237 then return 237
elseif n == 238 then return 238
elseif n == 239 then return 239
elseif n == 240 then return 240
elseif n == 241 then return 241
elseif n == 242 then return 242
elseif n == 243 then return 243
elseif n == 244 then return 244
elseif n == 245 then return 245
elseif n == 246 then return 246
elseif n == 247 then return 247
elseif n == 248 then return 248
elseif n == 249 then return 249
elseif n == 250 then return 250
elseif n == 251 then return 251
elseif n == 252 then return 252
elseif n == 253 then return 253
elseif n == 254 then return 254
elseif n == 255 then return 255
elseif n == 256 then return 256
elseif n == 257 then return 257
elseif n == 258 then return 258
elseif n == 259 then return 259
elseif n == 260 then return 260
elseif n == 261 then return 261
elseif n == 262 then return 262
elseif n == 263 then return 263
elseif n == 264 then return 264
elseif n == 265 then return 265
elseif n == 266 then return 266
elseif n == 267 then return 267
elseif n == 268 then return 268
elseif n == 269 then return 269
elseif n == 270 then return 270
elseif n == 271 then return 271
elseif n == 272 then return 272
elseif n == 273 then return 273
elseif n == 274 then return 274
elseif n == 275 then return 275
elseif n == 276 then return 276
elseif n == 277 then return 277
elseif n == 278 then return 278
elseif n == 279 then return 279
elseif n == 280 then return 280
elseif n == 281 then return 281
elseif n == 282 then return 282
elseif n == 283 then return 283
elseif n == 284 then return 284
elseif n == 285 then return 285
elseif n == 286 then return 286
elseif n == 287 then return 287
elseif n == 288 then return 288
elseif n == 289 then return 289
elseif n == 290 then return 290
elseif n == 291 then return 291
elseif n == 292 then return 292
elseif n == 293 then return 293
elseif n == 294 then return 294
elseif n == 295 then return 295
elseif n == 296 then return 296
elseif n == 297 then return 297
elseif n == 298 then return 298
elseif n == 299 then return 299
    else return -1 end
end
print(pick(1), pick(150), pick(299), pick(1000))

local o2 = { log = function(...) return select("#", ...) end }
print(o2:log(1))
print(io.stdout.write ~= nil)
