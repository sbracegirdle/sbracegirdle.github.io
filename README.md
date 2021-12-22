
# Let's Build

Source code for:

https://letsbuild.cloud

Checklist

- [] Syntax highlighting
- [] Benchmark site
- [] Link-tree-like site for bracegirdle.me.

- [x] Add OG tags
- [x] Add/remove about page
- [x] Replace / fix menu button on mobile

## Installation and pre-requisites

Install tailwind globally:

```sh
npm i -g tailwindcss
```

## How to build

Generate new CSS:

```sh
npx tailwindcss -i css/input.css -o css/output.css
```

## How to deploy

Commit and push, it will deploy automatically via GitHub Pages.
