# Formatting Verbs

The `format()` builtin understands the following verbs. Unless stated otherwise, format specifiers behave like their standard counterparts in languages that follow printf-style conventions.

## General

```
%v  default representation for the value
%T  runtime type name
%%  literal percent sign (consumes no value)
```

## Boolean

```
%t  the word true or false
```

## Integer

```
%b  base 2
%c  the character represented by the corresponding Unicode code point
%d  base 10
%o  base 8
%O  base 8 with 0o prefix
%q  a single-quoted character literal, escaped safely
%x  base 16, with lower-case letters for a-f
%X  base 16, with upper-case letters for A-F
%U  Unicode format: U+1234; same as "U+%04X"
```

## Float

```
%b  binary exponent form, e.g. -123456p-78
%e  scientific notation, e.g. -1.234456e+78
%E  scientific notation, e.g. -1.234456E+78
%f  decimal point but no exponent, e.g. 123.456
%F  synonym for %f
%g  %e for large exponents, %f otherwise
%G  %E for large exponents, %F otherwise
%x  hexadecimal notation, e.g. -0x1.23abcp+20
%X  upper-case hexadecimal notation, e.g. -0X1.23ABCP+20
```

## String and Bytes

```
%s  uninterpreted bytes of the string or slice
%q  a double-quoted string safely escaped
%x  base 16, lower-case, two characters per byte
%X  base 16, upper-case, two characters per byte
```

## Default %v output

```
Bool:    %t
Int:     %d
Float:   %g
String:  %s
```

Arrays print as `[elem0 elem1 ...]` and maps print as `{key1:value1 ...}`.

## Width and Precision

Width precedes the verb as a decimal number. Precision comes after a period. Either field can use `*` to read the value from the next argument.

```
%f     default width, default precision
%9f    width 9, default precision
%.2f   default width, precision 2
%9.2f  width 9, precision 2
%9.f   width 9, precision 0
```

For strings and bytes, precision limits the number of runes (or bytes when using `%x`/`%X`). For floats, precision controls digits after the decimal except for `%g`/`%G`, where it sets the maximum number of significant digits.

## Flags

```
+   always print a sign for numeric values; guarantees ASCII-only `%q`
-   pad with spaces on the right (left-justify the field)
#   alternate format (0b, 0o, 0x prefixes, raw strings for `%q`, decimal point)
' ' leave a space for an elided sign in numbers; add spaces between bytes for %x
0   pad with leading zeros; for numbers this comes after the sign
```

Flags ignored by a verb have no effect.
