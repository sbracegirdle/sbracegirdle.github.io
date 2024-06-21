---
layout: ../layouts/Post.astro
title: How to re-organise commits for code review
date: 2024-06-21
tags: git commits code_review
author: Simon Bracegirdle
description: How I clean up commits by squashing and creating new logical commits before opening for code review.
---

If you're like me, you'll often commit work as you go. This is a good idea in general because you're effectively able to backup your work, have breakpoints to compare against, and a restore point in case of a mistake or change in direction. But, when you finish the particular change you're working on, the commit history can end up looking like a bit of a mess:

```
abcdef My feature; WIP
abcdef My feature; Oops broke something, fxied it
abcdef My feature; More fixes
abcdef My feature; Even more fixes
```

This isn't so nice for people reviewing your code, as the commit history is no longer a logical reflection of the change as a whole, and doesn't help us understand what changed and why.

Sometimes this isn't a big deal, like if it's a small change, since we can always squash on merge if using GitHub or another platform with that feature. But for bigger changes it can be a good idea to clean up your commits before opening for review.

This is also a good idea if you're continuing to build other changes on top of the first one, since it'll make it easier to resolve merge conflicts, or rebase in case we add further commits to the first change after starting work on the second.

## Step 1. Squashing

Experienced git users will be familiar with this, and it's the first step in the process for combining commits that aren't of valid in the history.

Returning to the example before, if we want to squash the last four commits, we start with:

```sh
git rebase -i HEAD~4
```

This will start an interactive rebase with the last four commits from the current HEAD position.

In the interactive rebase file, change 'pick' to 'squash' for every commit *except the first*, then save and exit.

For example:

```sh
pick ...
squash ...
squash ...
squash ...
```

Take care not to exclude a commit, because then it'll be difficult to get it back.

Save and continue.


## Step 2. Creating new logical commits

If we now want to break up the squashed commit into logical commits, start a rebase again:

```sh
git rebase -i HEAD~1
```

In the interactive rebase file, change `pick` to `edit` for the commit to split, then save and exit.

Reset HEAD to previous commit without changing the working directory, this will allow us to re-add and commit our files:

```sh
git reset HEAD^
```

Now we can add and commit files in our change as we like to form new logical commits:

```sh
git add <file_or_part_of_it>
git commit -m "Feature X; Added new entities and migrations for the feature"

git add <file_or_part_of_it>
git commit -m "Feature X; Added new services because <reason>"
```

Then after finishing, continue the rebase:

```sh
git rebase --continue
```

## Step 3. Force push and open PR

Assuming we now have commit history in a form that makes sense and will make reviewing easier, we can force push the branch (assuming it's already pushed to remote, otherwise no need to) and open a PR:

```sh
git push --force
```

## Summary

Working with git rebase is difficult and complex to understand, but it's a powerful tool and can help make ciommit history more readable and reviewable for others.

