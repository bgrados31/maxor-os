{ lib, buildGoModule }:

# La pantalla completa de Maxor (maxor-tui). Es un programa aparte, en Go: la CLI
# (`maxor`, en bash) sigue haciendo todo el trabajo y la pantalla solo la llama
# con --json. El código está en tui/; `go test` corre en cada compilación.
let
  version = lib.removeSuffix "\n" (builtins.readFile ../VERSION);
in
buildGoModule {
  pname = "maxor-tui";
  inherit version;
  src = lib.cleanSource ../tui;
  vendorHash = "sha256-2ZMkFBbvQQ/hDavxKgO6XMz7I6zqhwKqd0AuuXCX8zU=";
  env.CGO_ENABLED = "0"; # binario estático, sin depender de gcc ni de libc
  # Bubble Tea pregunta a la terminal por su color de fondo en su init() y espera hasta
  # 5 s la respuesta (termenv.OSCTimeout es una constante): en una terminal que no
  # contesta (consola de Linux, algunos SSH) la pantalla tardaría 5 s en abrirse. Maxor
  # usa solo colores explícitos, así que se quita esa consulta. Si una versión nueva
  # de Bubble Tea cambia esa línea, `--replace-fail` detiene la compilación.
  preBuild = ''
    substituteInPlace vendor/github.com/charmbracelet/bubbletea/tea_init.go \
      --replace-fail "_ = lipgloss.HasDarkBackground()" "_ = lipgloss.HasDarkBackground // (consulta omitida por Maxor)"
  '';
  # el módulo termina en «tui»: el binario se instala con su nombre real
  postInstall = "mv $out/bin/tui $out/bin/maxor-tui";
  ldflags = [ "-s" "-w" "-X main.version=${version}" ];
  meta = {
    description = "Pantalla completa de Maxor OS";
    mainProgram = "maxor-tui";
    license = lib.licenses.mit;
  };
}
