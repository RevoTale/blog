# UI Link Behavior

## Note Title Links

- Note title links in the feed keep the same text color for both unvisited and visited states.
- The status dot is displayed next to the `Open full note` action link.
- The dot is green for unvisited and hidden for visited.
- The visited/unvisited state is implemented with `:link::before` on `.note-open-link`.
