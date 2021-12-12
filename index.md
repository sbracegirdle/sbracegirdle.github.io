---
title: Let's Build
---

# Let's Build

<ul>
  {% for post in site.posts %}
  <a href="{{ post.url }}">
    <li>
      <h3>{{ post.title }}</h3>
      <p>{{ post.description }}</p>
    </li>
  </a>
  {% endfor %}
</ul>

