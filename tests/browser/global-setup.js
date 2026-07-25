// Hard guard: the browser tests run headless, always. `--headed` and
// `--debug` on the command line override the config's `headless: true`, so
// check the *resolved* config and refuse to run if anything turned it off.
// A stray headed run steals focus on the host machine, which is exactly what
// this suite is not allowed to do.
export default function globalSetup(config) {
  const headed = config.projects.filter((p) => p.use?.headless === false);
  if (headed.length > 0) {
    const names = headed.map((p) => p.name || "(unnamed)").join(", ");
    throw new Error(
      `Refusing to run: headed mode requested for project(s) ${names}. ` +
        `These tests must stay headless — drop --headed/--debug. ` +
        `To watch a run instead, use the HTML report: npx playwright show-report.`,
    );
  }
}
