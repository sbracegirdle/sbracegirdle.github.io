/* ══════════════════════════════════════════════════════════════
   arcade — a side-scrolling ASCII shooter for the homepage panel.

   The one piece of JavaScript on this site. No build step, no
   dependencies, no requests of its own: the playfield is a grid of
   characters written into the `pre` the generator already emitted,
   and every glyph takes a class from theme.css.

   Two modes. By default the ship flies itself, so the panel is an
   attract loop nobody has to touch. Click it, or press play, and the
   keyboard and pointer are yours until Escape, the stop button, a
   click outside, a Tab away, or scrolling the panel out of view.

   Three things worth knowing before changing anything here:

   - The grid is measured, not assumed. Rows come from
     --arcade-rows in theme.css and columns from whatever fits the
     panel at the visitor's actual monospace font, so the same code
     runs about ninety columns wide on a desktop and thirty on a
     phone.
   - Only cell contents are animated. Nothing moves in CSS, no
     element is created per frame, and a frame identical to the last
     one is not written to the DOM at all.
   - It stops when nobody is watching, or when told to: off-screen,
     in a background tab, under prefers-reduced-motion until the
     visitor presses play, and on the pause button, which is there
     because something that starts on its own and runs past five
     seconds has to be stoppable.
   ══════════════════════════════════════════════════════════════ */
(() => {
  "use strict";

  const root = document.querySelector(".arcade");
  if (!root) return;
  const screen = root.querySelector(".arcade-screen");
  const frame = root.querySelector(".arcade-frame");
  const play = root.querySelector(".arcade-play");
  const pause = root.querySelector(".arcade-pause");
  if (!screen || !frame || !play || !pause) return;

  const hud = {
    score: root.querySelector(".arcade-score"),
    wave: root.querySelector(".arcade-wave"),
    lives: root.querySelector(".arcade-lives"),
    mode: root.querySelector(".arcade-mode"),
  };

  // ── The grid ────────────────────────────────────────────────
  // Paint classes, in the order the ids below index them. The hues
  // are the sitewide ones doing their sitewide jobs: foam is you,
  // love is what is shooting at you, gold is the shot and the
  // score, iris the second attacker, rose the explosion. Every
  // entity also has its own glyph, so none of that is carried by
  // colour alone.
  const PAINT = ["", "a-star", "a-ship", "a-shot", "a-foe", "a-weave", "a-boom", "a-hint"];
  const NONE = 0, STAR = 1, SHIP = 2, SHOT = 3, FOE = 4, WEAVE = 5, BOOM = 6, HINT = 7;

  let cols = 0, rows = 0, cellW = 8, cellH = 16;
  let chars = [], paint = new Uint8Array(0);

  // The panel's real geometry, in the visitor's real font. Three
  // fonts deep in the stack may be missing, so a column count worked
  // out from a nominal character width would be wrong on most
  // machines: measure a run of glyphs instead and divide.
  function measure() {
    const probe = document.createElement("span");
    probe.textContent = "0".repeat(50);
    probe.style.cssText = "position:absolute;visibility:hidden;white-space:pre";
    frame.appendChild(probe);
    cellW = probe.getBoundingClientRect().width / 50 || 8;
    probe.remove();

    const style = getComputedStyle(frame);
    cellH = parseFloat(style.lineHeight) || 16;
    const wasCols = cols, wasRows = rows;
    rows = Math.max(7, parseInt(style.getPropertyValue("--arcade-rows"), 10) || 11);
    cols = Math.max(20, Math.min(140, Math.floor(frame.clientWidth / cellW)));
    if (cols === wasCols && rows === wasRows) return false;

    chars = new Array(cols * rows).fill(" ");
    paint = new Uint8Array(cols * rows);
    // Only on a real change of size. A resize callback fires more than
    // once for reasons that have nothing to do with the grid, and
    // re-seeding on each of them redrew the starfield under a visitor
    // who had asked for no motion at all.
    seedStars();
    clampToGrid();
    return true;
  }

  function put(x, y, ch, cls) {
    x = Math.round(x);
    y = Math.round(y);
    if (x < 0 || x >= cols || y < 0 || y >= rows) return;
    chars[y * cols + x] = ch;
    paint[y * cols + x] = cls;
  }

  function write(x, y, text, cls) {
    for (let i = 0; i < text.length; i++) put(x + i, y, text[i], cls);
  }

  const ENTITIES = { "&": "&amp;", "<": "&lt;", ">": "&gt;" };
  const esc = (s) => s.replace(/[&<>]/g, (c) => ENTITIES[c]);

  // One string per frame, runs of a colour wrapped in a span and
  // everything else left as text. A row is cut at its last non-blank
  // cell: trailing spaces are most of the grid and none of them are
  // visible.
  let painted = "";
  function draw() {
    let out = "";
    for (let y = 0; y < rows; y++) {
      const row = y * cols;
      let end = cols - 1;
      while (end >= 0 && chars[row + end] === " " && paint[row + end] === NONE) end--;

      let run = "", cls = NONE;
      for (let x = 0; x <= end; x++) {
        if (paint[row + x] !== cls) {
          out += wrap(run, cls);
          run = "";
          cls = paint[row + x];
        }
        run += chars[row + x];
      }
      out += wrap(run, cls);
      if (y < rows - 1) out += "\n";
    }
    if (out === painted) return;
    painted = out;
    frame.innerHTML = out;
  }

  function wrap(run, cls) {
    if (!run) return "";
    return cls === NONE ? esc(run) : `<span class="${PAINT[cls]}">${esc(run)}</span>`;
  }

  // ── The game ────────────────────────────────────────────────
  const TICK = 55;           // ms per step; the world moves in whole cells
  const SHIP_DY = 0.62;      // cells per step, vertical
  const SHIP_DX = 0.5;
  const SHOT_VX = 1.9;
  const BOLT_VX = 1.15;
  const FIRE_COOL = 5;       // steps between shots
  const BOOM_ART = ["#", "*", "+", "."];

  // Attackers, all facing left. hp and speed carry the difficulty;
  // the glyph carries which is which.
  const FOES = [
    { art: "<-", hp: 1, vx: 0.62, cls: FOE, score: 10, shoots: 0 },
    { art: "{o}", hp: 1, vx: 0.4, cls: WEAVE, score: 25, shoots: 0, weave: 2.2 },
    { art: "<[#]", hp: 3, vx: 0.24, cls: FOE, score: 50, shoots: 0.035 },
  ];

  const ship = { x: 3, y: 5, cool: 0, blink: 0 };
  const stars = [];
  let shots = [], bolts = [], foes = [], booms = [];
  let score = 0, wave = 1, lives = 3, kills = 0, spawn = 12, over = 0, tick = 0;

  const clamp = (n, lo, hi) => (n < lo ? lo : n > hi ? hi : n);

  function seedStars() {
    stars.length = 0;
    const many = Math.max(8, Math.round((cols * rows) / 26));
    for (let i = 0; i < many; i++) {
      stars.push({
        x: Math.random() * cols,
        y: Math.floor(Math.random() * rows),
        v: 0.12 + Math.random() * 0.5,
      });
    }
  }

  function clampToGrid() {
    ship.y = clamp(ship.y, 0, rows - 1);
    ship.x = clamp(ship.x, 0, Math.floor(cols * 0.55));
    foes = foes.filter((f) => f.y < rows);
    shots = shots.filter((s) => s.x < cols);
    bolts = bolts.filter((b) => b.y < rows);
  }

  function reset() {
    shots = [];
    bolts = [];
    foes = [];
    booms = [];
    score = 0;
    wave = 1;
    lives = 3;
    kills = 0;
    spawn = 12;
    over = 0;
    ship.x = 3;
    ship.y = (rows - 1) / 2;
    ship.cool = 0;
    ship.blink = 0;
    // The board opens with attackers already on it. Spawning from an
    // empty screen took a second and a half to look like anything, and
    // that first second is the whole of what a visitor sees before
    // scrolling on.
    for (const [t, atX, atY] of [[0, 0.55, 0.2], [1, 0.75, 0.7], [2, 0.92, 0.5]]) {
      const y = clamp(Math.round((rows - 1) * atY), 0, rows - 1);
      foes.push({ t, x: cols * atX, y, y0: y, hp: FOES[t].hp, phase: 0 });
    }
    showHud();
  }

  function showHud() {
    if (hud.score) hud.score.textContent = String(score).padStart(5, "0");
    if (hud.wave) hud.wave.textContent = String(wave);
    if (hud.lives) hud.lives.textContent = String(Math.max(0, lives));
  }

  // What the ship is asked to do this step, whoever is flying it.
  function playerDrive() {
    let dy = 0, dx = 0;
    if (keys.has("up")) dy -= 1;
    if (keys.has("down")) dy += 1;
    if (keys.has("left")) dx -= 1;
    if (keys.has("right")) dx += 1;
    // The pointer wins while it is over the panel: it is an absolute
    // position, so it steers rather than nudges.
    if (dy === 0 && pointer.y !== null) dy = clamp((pointer.y - ship.y) / SHIP_DY, -1.4, 1.4);
    if (dx === 0 && pointer.x !== null) dx = clamp((pointer.x - ship.x) / SHIP_DX, -1.4, 1.4);
    return { dy, dx, fire: keys.has("fire") || pointer.fire };
  }

  // The autopilot. It aims at the nearest attacker ahead of it,
  // slides out of the lane of anything about to hit it, and fires
  // when it is lined up. Deliberately not perfect: it commits to a
  // target for a few steps at a time, which is what lets it lose.
  let botTarget = null, botHold = 0;
  function botDrive() {
    if (botHold > 0) botHold--;
    if (!botTarget || botHold === 0 || foes.indexOf(botTarget) === -1) {
      botTarget = null;
      for (const f of foes) {
        if (f.x > ship.x + 3 && (!botTarget || f.x < botTarget.x)) botTarget = f;
      }
      botHold = 3 + Math.floor(Math.random() * 5);
    }

    let want = botTarget ? botTarget.y : ship.y;
    let panic = false;
    for (const b of bolts) {
      if (b.x > ship.x && b.x - ship.x < 13 && Math.abs(b.y - ship.y) < 1.2) panic = true;
    }
    for (const f of foes) {
      if (f.x > ship.x && f.x - ship.x < 7 && Math.abs(f.y - ship.y) < 1.2) panic = true;
    }
    if (panic) want = ship.y < rows / 2 ? ship.y + 2.5 : ship.y - 2.5;

    const gap = want - ship.y;
    return {
      dy: clamp(gap, -1, 1),
      dx: 0,
      // It lets them come to it. Firing the moment a target appears
      // killed everything against the right-hand edge and left the
      // panel looking empty from the outside.
      fire: !panic && botTarget !== null && botTarget.x < cols * 0.78 && Math.abs(gap) < 0.9,
    };
  }

  function damage(f, i) {
    f.hp--;
    if (f.hp > 0) {
      booms.push({ x: f.x, y: f.y, life: 2 });
      return;
    }
    const kind = FOES[f.t];
    for (let k = 0; k < kind.art.length; k++) booms.push({ x: f.x + k, y: f.y, life: 4 });
    foes.splice(i, 1);
    score += kind.score;
    kills++;
    if (kills % 12 === 0 && wave < 9) wave++;
    showHud();
  }

  function hitShip() {
    if (ship.blink > 0 || over > 0) return;
    for (let k = 0; k < 3; k++) booms.push({ x: ship.x + k, y: ship.y, life: 4 });
    lives--;
    ship.blink = 26;
    showHud();
    if (lives <= 0) over = 36;
  }

  // Cells [ax, bx] on row y, swept: a shot moving two cells a step
  // must not pass through a two-cell attacker without touching it.
  const overlaps = (ax, bx, x, width) => bx >= x && ax <= x + width - 1;

  function step() {
    tick++;

    for (const s of stars) {
      s.x -= s.v;
      if (s.x < 0) {
        s.x = cols - 1;
        s.y = Math.floor(Math.random() * rows);
      }
    }
    for (let i = booms.length - 1; i >= 0; i--) if (--booms[i].life <= 0) booms.splice(i, 1);

    // The beat after the last life, so the explosion is seen before
    // the board is cleared. Player mode hands the ship back to the
    // autopilot rather than sitting on a dead screen.
    if (over > 0) {
      if (--over === 0) {
        reset();
        if (mode === "you") setMode("bot");
      }
      compose();
      return;
    }

    const drive = mode === "you" ? playerDrive() : botDrive();
    ship.y = clamp(ship.y + drive.dy * SHIP_DY, 0, rows - 1);
    ship.x = clamp(ship.x + drive.dx * SHIP_DX, 0, Math.floor(cols * 0.55));
    if (ship.cool > 0) ship.cool--;
    if (ship.blink > 0) ship.blink--;
    if (drive.fire && ship.cool === 0) {
      shots.push({ x: ship.x + 3, y: Math.round(ship.y) });
      ship.cool = FIRE_COOL;
    }

    for (let i = shots.length - 1; i >= 0; i--) {
      const s = shots[i];
      const from = Math.round(s.x);
      s.x += SHOT_VX;
      const to = Math.round(s.x);
      let spent = to >= cols;
      for (let j = foes.length - 1; j >= 0 && !spent; j--) {
        const f = foes[j];
        if (Math.round(f.y) !== s.y) continue;
        if (!overlaps(from, to, Math.round(f.x), FOES[f.t].art.length)) continue;
        damage(f, j);
        spent = true;
      }
      if (spent) shots.splice(i, 1);
    }

    for (let i = bolts.length - 1; i >= 0; i--) {
      const b = bolts[i];
      const from = Math.round(b.x);
      b.x -= BOLT_VX;
      const to = Math.round(b.x);
      if (to < 0) {
        bolts.splice(i, 1);
        continue;
      }
      if (Math.round(b.y) === Math.round(ship.y) && overlaps(to, from, Math.round(ship.x), 3)) {
        hitShip();
        bolts.splice(i, 1);
      }
    }

    for (let i = foes.length - 1; i >= 0; i--) {
      const f = foes[i];
      const kind = FOES[f.t];
      f.x -= kind.vx * (1 + wave * 0.06);
      if (kind.weave) f.y = clamp(f.y0 + Math.sin(tick * 0.16 + f.phase) * kind.weave, 0, rows - 1);
      if (kind.shoots && f.x < cols - 2 && Math.random() < kind.shoots) {
        bolts.push({ x: f.x - 1, y: Math.round(f.y) });
      }
      if (f.x < -kind.art.length) {
        foes.splice(i, 1);
        continue;
      }
      if (
        Math.round(f.y) === Math.round(ship.y) &&
        overlaps(Math.round(f.x), Math.round(f.x) + kind.art.length - 1, Math.round(ship.x), 3)
      ) {
        hitShip();
        damage(f, i);
      }
    }

    // Mostly scouts, some weavers, the occasional gunship. The mix is fixed
    // and the wave only shortens the gap between spawns: a wave that changed
    // both at once got unreadable three waves in, a screen of gunships all
    // firing at a ship that never had a clear lane.
    if (--spawn <= 0) {
      const roll = Math.random();
      const t = roll < 0.56 ? 0 : roll < 0.86 ? 1 : 2;
      const y = 1 + Math.floor(Math.random() * Math.max(1, rows - 2));
      foes.push({ t, x: cols + 1, y, y0: y, hp: FOES[t].hp, phase: Math.random() * 6.28 });
      spawn = Math.max(8, 26 - wave * 2) + Math.floor(Math.random() * 6);
    }

    compose();
  }

  // Everything that is on the screen this step, back to front.
  function compose() {
    chars.fill(" ");
    paint.fill(NONE);

    for (const s of stars) put(s.x, s.y, s.v > 0.4 ? "." : "'", STAR);
    for (const f of foes) write(f.x, f.y, FOES[f.t].art, FOES[f.t].cls);
    for (const s of shots) put(s.x, s.y, "-", SHOT);
    for (const b of bolts) put(b.x, b.y, "o", FOE);
    // Blinking through the invulnerable beat after a hit: three steps
    // on, three off, which is 3 Hz — the rate above which flashing
    // stops being decoration and starts being a hazard.
    if (ship.blink === 0 || Math.floor(ship.blink / 3) % 2 === 0) {
      write(ship.x, ship.y, (tick % 2 ? "-" : "~") + "=>", SHIP);
    }
    for (const b of booms) put(b.x, b.y, BOOM_ART[BOOM_ART.length - b.life] || ".", BOOM);

    if (over > 0) banner("game over");
    else if (mode === "bot" && tick % 44 < 30) {
      write(Math.max(0, Math.floor((cols - 13) / 2)), rows - 1, "click to play", HINT);
    }
  }

  function banner(text) {
    write(Math.max(0, Math.floor((cols - text.length) / 2)), Math.floor(rows / 2) - 1, text, HINT);
  }

  // ── Who is flying, and what wakes the loop ──────────────────
  const keys = new Set();
  const pointer = { x: null, y: null, fire: false };
  let mode = "bot";

  // The label says which state the button is in, so it carries no
  // aria-pressed: a toggle does one or the other, and "stop, pressed"
  // announces the opposite of what it means.
  function setMode(next) {
    if (mode === next) return;
    mode = next;
    const playing = next === "you";
    root.classList.toggle("playing", playing);
    play.textContent = playing ? "stop" : "play";
    if (hud.mode) hud.mode.textContent = playing ? "you" : "auto";
    keys.clear();
    pointer.x = pointer.y = null;
    pointer.fire = false;
    reset();
    if (playing) {
      setHalted(false);
      run();
    }
  }

  // Enter belongs to whatever has focus, which while playing is the
  // stop button — the one control on screen saying how to get out.
  // Space fires, Escape leaves.
  const KEYS = {
    arrowup: "up", w: "up", k: "up",
    arrowdown: "down", s: "down", j: "down",
    arrowleft: "left", a: "left", h: "left",
    arrowright: "right", d: "right", l: "right",
    " ": "fire",
  };
  const named = (e) => KEYS[e.key.toLowerCase()];

  play.addEventListener("click", () => setMode(mode === "you" ? "bot" : "you"));

  // A click on the screen takes the controls and fires in the same
  // motion. Focus follows it to the button so the keys work too, and
  // so there is something visible holding focus.
  //
  // preventDefault is doing more here than stopping a stray text
  // selection. A press on a non-focusable element moves focus to the
  // body by default, and that arrived as a focusout of the button we
  // had just focused — so every click on the panel took the controls
  // and handed them straight back.
  //
  // A finger never takes the controls on the way down. pointerdown
  // fires at the start of a scroll flick, and taking the controls
  // there put `touch-action: none` under the thumb — after one flick,
  // a 250px band of the home page had stopped scrolling. A touch has
  // to land, stay put, and lift.
  let tapFrom = null;
  screen.addEventListener("pointerdown", (e) => {
    if (e.pointerType === "touch") {
      tapFrom = { x: e.clientX, y: e.clientY };
      if (mode === "you") {
        aim(e);
        pointer.fire = true;
      }
      return;
    }
    e.preventDefault();
    if (mode !== "you") setMode("you");
    play.focus({ preventScroll: true });
    aim(e);
    pointer.fire = true;
  });
  screen.addEventListener("pointerup", (e) => {
    if (e.pointerType !== "touch" || !tapFrom) return;
    const still = Math.abs(e.clientX - tapFrom.x) < 10 && Math.abs(e.clientY - tapFrom.y) < 10;
    tapFrom = null;
    // No focus() on this path. A touch still fires a compatibility
    // mousedown, whose default action moves focus to the body — which
    // arrives as a focusout of the button we would just have focused,
    // and hands the controls straight back.
    if (still && mode !== "you") setMode("you");
  });
  screen.addEventListener("pointermove", (e) => mode === "you" && aim(e));
  screen.addEventListener("pointerleave", () => {
    pointer.x = pointer.y = null;
    pointer.fire = false;
  });
  addEventListener("pointerup", () => (pointer.fire = false));

  function aim(e) {
    const box = frame.getBoundingClientRect();
    pointer.y = clamp((e.clientY - box.top) / cellH - 0.5, 0, rows - 1);
    pointer.x = clamp((e.clientX - box.left) / cellW - 1.5, 0, cols);
  }

  // Keys are only ever intercepted while the panel has the
  // controls — a visitor scrolling past with the arrow keys should
  // never find the page stuck. Escape is the way out, and Enter and
  // Space fire rather than re-activating the focused button.
  addEventListener("keydown", (e) => {
    if (e.key === "Escape" && mode === "you") {
      setMode("bot");
      e.preventDefault();
      return;
    }
    if (mode !== "you" || e.metaKey || e.ctrlKey || e.altKey) return;
    const key = named(e);
    if (!key) return;
    keys.add(key);
    e.preventDefault();
  });
  addEventListener("keyup", (e) => {
    const key = named(e);
    if (key) keys.delete(key);
  });

  // Leaving by any route hands the ship back: Tab out of the panel,
  // or click anywhere else on the page.
  //
  // Only when focus lands somewhere else, though. Focus leaving for
  // nowhere in particular — which is what a compatibility mousedown on
  // the screen does — isn't someone leaving the panel, and the click
  // below already covers a press that really is outside it.
  root.addEventListener("focusout", (e) => {
    if (mode === "you" && e.relatedTarget && !root.contains(e.relatedTarget)) setMode("bot");
  });
  addEventListener("pointerdown", (e) => {
    if (mode === "you" && !root.contains(e.target)) setMode("bot");
  });
  addEventListener("blur", () => keys.clear());

  // ── The loop ────────────────────────────────────────────────
  // Fixed steps, drained from real elapsed time, so the game runs at
  // the same speed on any display and a frame the browser skipped
  // costs a step rather than a stutter.
  // Reduced motion is the panel starting out halted, rather than a
  // second kind of stopped: one flag, one button, and the button's
  // label is true the moment the page loads. Pressing "resume" is the
  // same ask as pressing play, so it starts the loop either way.
  const calm = matchMedia("(prefers-reduced-motion: reduce)");
  let onScreen = true, halted = calm.matches, frameID = 0, prev = 0, owed = 0;

  const wanted = () => onScreen && !halted && !document.hidden;

  // The pause button is not a nicety. Something that starts by itself,
  // runs longer than five seconds and sits beside text you are trying
  // to read has to be stoppable (WCAG SC 2.2.2), and "stop" next to it
  // doesn't count — that hands the ship back to the autopilot, which
  // keeps flying. Halting drops the controls too, so nothing is left
  // holding the arrow keys over a frozen panel.
  pause.addEventListener("click", () => setHalted(!halted));

  function setHalted(next) {
    if (halted === next) return;
    halted = next;
    pause.textContent = halted ? "resume" : "pause";
    if (halted) {
      setMode("bot");
      stop();
    } else {
      run();
    }
  }

  function loop(now) {
    if (!wanted()) {
      frameID = 0;
      return;
    }
    frameID = requestAnimationFrame(loop);
    const dt = Math.min(now - prev, 250);
    prev = now;
    owed += dt;
    while (owed >= TICK) {
      owed -= TICK;
      step();
    }
    draw();
  }

  function run() {
    if (frameID || !wanted()) return;
    prev = performance.now();
    owed = 0;
    frameID = requestAnimationFrame(loop);
  }

  function stop() {
    if (frameID) cancelAnimationFrame(frameID);
    frameID = 0;
  }

  new ResizeObserver(() => {
    if (!measure()) return;
    // Nothing has been drawn yet and nothing is going to be: the panel
    // is still showing the generator's frame, and a resize is no
    // reason to take it away.
    if (!painted && !wanted()) return;
    painted = "";
    compose();
    draw();
  }).observe(screen);

  // Scrolling the panel out of sight gives the ship back as well as
  // stopping the loop. Otherwise the arrow keys stayed captured by a
  // panel the visitor could no longer see, and the page had quietly
  // stopped scrolling with nothing on screen to explain why.
  new IntersectionObserver((entries) => {
    onScreen = entries[entries.length - 1].isIntersecting;
    if (!onScreen && mode === "you") setMode("bot");
    onScreen ? run() : stop();
  }).observe(root);

  document.addEventListener("visibilitychange", () => (document.hidden ? stop() : run()));
  calm.addEventListener("change", () => setHalted(calm.matches));

  measure();
  reset();
  pause.textContent = halted ? "resume" : "pause";
  // Under prefers-reduced-motion the loop won't start, and the first
  // frame the script draws would be the only one the visitor ever
  // sees — so leave the still frame the generator drew, which is a
  // better picture of the game than a cold opening board.
  if (wanted()) {
    compose();
    draw();
    run();
  }
})();
