### Alternative Delimiters

When your expression needs to contain the `}}` character sequence (such as when
working with Go template syntax), use alternative delimiters `${%` and `%}`
instead:

```yaml
config:
  # This would fail with standard delimiters due to }} in the string
  message: ${% "Updated image to {{.tag}}" %}
```

The above example will be evaluated as follows:

```yaml
config:
  message: Updated image to {{.tag}}
```

You can mix both delimiter types in the same value, and all expressions will be
evaluated:

```yaml
config:
  # Both expressions are evaluated
  message: ${{ ctx.stage }} deployed ${% "{{.tag}}" %}
```

:::warning
Within a single expression, you cannot use the `}}` sequence with standard
delimiters `${{ }}`, nor can you use the `%}` sequence with alternative
delimiters `${% %}`. Choose the appropriate delimiter for each expression based
on which characters you need to include.
:::

:::tip
Use standard delimiters `${{ }}` by default. Only switch to alternative
delimiters `${% %}` for expressions that need to contain the `}}` sequence.
:::
