
# Let's Build

Source code for:

https://letsbuild.cloud

Checklist

- [] Syntax highlighting
- [] Lighthouse scan site
- [] Link-tree-like site for bracegirdle.me.
- [] Site metrics
- [] OG Images for posts

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

## Download image from unsplash

```sh
curl -L "https://source.unsplash.com/1600x900/?team,collaborate" > img/team1.jpg
```

Copy image and resize as thumbnail:

```sh
cp img/mypost.jpg img/mypost-thumb.jpg
mogrify -resize 400x img/mypost-thumb.jpg
```
