---
simpleA: "valueA"
simple-b: "valueB"
simple-C: valueC
long-without-quotes: A long value
long-with-simple-quotes: 'A long value'
long-with-double-quotes: "A long value"
integer: 10
float: 10.50
boolean: true
array-inline: ["value1", "value2"]
array-block:
- value1
- value2
array-single-value: ["value1"]
object:
- subpropertyA: valueA
- subpropertyB: 10
---

# Complex Notes Suite

<!--
A comment that is not considered as part of the notes
-->

## Note: Markdown in Markdown

`@source: https://en.wikipedia.org/wiki/Markdown`

Example:

```md
# Markdown

## History

John Gruber created Markdown in 2004.
```

> Basic inclusion of markdown documents is supported.


## Cheatsheet: How to include HTML in Markdown

`#html`

Use HTML directly:

```md
Most inline <abbr title="Hypertext Markup Language">HTML</abbr> tags are supported.
```


## Note: A

`@source: https://www.markdownguide.org/basic-syntax/#headings`

`#tag-a`

This is a **parent note**.

### Note: B

`#tag-b1` `#tag-b2`

This is **child and parent note**.

#### Note: C

`@source: https://www.markdownguide.org/basic-syntax/#headings`

This is **child and parent note**.

##### Note: D

`#tag-d`

This is **child and parent note**.

###### Note: E

This is **childnote**.


## TODO: List

* [x] Task A `@priority: high` `#low-motivation` `#low-energy`
* [x] Task B `@priority: low`
* Task C
  * [x] Task C1 `@priority: medium` `#low-motivation`
  * [ ] Task C2
* Task D
  * Subtask D1
    * [ ] Implement Draft
  * Subtask D2
    * [ ] Implement Draft


## Note: Comments

Notes can use HTML comments.

<!-- a single line comment -->

HTML comments are simply ignored when rendering notes.

<!--
A multi-line
comment.
-->

Therefore, use them sparingly.


## Quote: Richly Annotated Quote

`#life` `#doing`
`@name: Christine Mason Miller`
`@occupation: author` `@nationality: American`

`#life-changing`
`#courage`

At any given moment, you have the power to say: this is not how the story is going to end.

> Be the change.
