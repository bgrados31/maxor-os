# Prueba de punta a punta del sistema de releases en máquinas virtuales de NixOS:
# un servidor que publica el canal (nginx, con ETag de verdad) y un cliente con el paquete
# `maxor`, hablando por red. Se ejecuta con
#
#   nix build .#checks.x86_64-linux.release-vm -L
#
# Necesita KVM (si no, tarda mucho). Usa claves de prueba creadas al compilar: ni la clave
# real de release ni la red de verdad intervienen. La parte de `apply` (git, etiqueta firmada)
# se prueba en tests/updates.bats; aquí, lo que solo se ve con un servidor y una red reales.
{ pkgs }:

let
  testKeys = pkgs.runCommand "maxor-test-keys" { nativeBuildInputs = [ pkgs.openssh ]; } ''
    mkdir -p $out
    ssh-keygen -q -t ed25519 -N "" -C test -f $out/good
    ssh-keygen -q -t ed25519 -N "" -C test -f $out/evil
    printf 'maxor-release namespaces="git,maxor-release" %s\n' "$(cut -d' ' -f1,2 $out/good.pub)" > $out/allowed_signers
  '';

  # La CLI de verdad, pero confiando en la clave de prueba y con un notify-send que anota.
  maxorTest = pkgs.callPackage ../../packages/maxor.nix {
    releaseKeys = "${testKeys}/allowed_signers";
    releaseUrl = "http://server";
    libnotify = pkgs.writeShellScriptBin "notify-send" ''echo "$*" >> /tmp/notifications'';
  };

  # Los scripts de publicación tal como están en el repositorio.
  releaseTools = pkgs.runCommand "maxor-release-tools" { } ''
    mkdir -p $out/scripts
    cp ${../../scripts/release-manifest.sh} $out/scripts/release-manifest.sh
    cp ${../../scripts/release-notes.sh} $out/scripts/release-notes.sh
    cp ${../../CHANGELOG.md} $out/CHANGELOG.md
    chmod +x $out/scripts/*.sh
    patchShebangs $out/scripts
  '';
in
pkgs.testers.runNixOSTest {
  name = "maxor-release";

  nodes.server = { pkgs, ... }: {
    services.nginx = {
      enable = true;
      virtualHosts.default = { default = true; root = "/srv/channel"; };
    };
    networking.firewall.allowedTCPPorts = [ 80 ];
    systemd.tmpfiles.rules = [ "d /srv/channel 0755 root root -" ];
    environment.systemPackages = with pkgs; [ jq openssh coreutils gawk gnused gnugrep ];
  };

  nodes.client = { ... }: {
    environment.systemPackages = [ maxorTest ];
  };

  testScript = ''
    import json

    start_all()
    server.wait_for_unit("nginx.service")
    client.wait_for_unit("multi-user.target")
    server.succeed("mkdir -p /root/keys && cp ${testKeys}/good ${testKeys}/evil /root/keys/ && chmod 600 /root/keys/*")

    def publish(version, seq, key="good", commit="a" * 40):
        """Escribe y firma el manifiesto del canal, como lo haría scripts/release.sh."""
        # nginx saca el ETag de la hora (en segundos) y el tamaño: sin esperar, dos cambios
        # del mismo tamaño en el mismo segundo tendrían el mismo ETag (y darían 304).
        server.succeed("sleep 1")
        server.succeed(f"""
          cd /srv/channel
          jq -n --arg v {version} --argjson s {seq} --arg c {commit} \
            '{{schema: 1, product: "maxor-os", version: $v, tag: ("v" + $v), commit: $c, sequence: $s,
              published: "2026-10-04T00:00:00Z", summary: "Test release", url: "https://example/x"}}' > manifest.json.new
          mv manifest.json.new manifest.json
          rm -f manifest.json.sig
          ssh-keygen -Y sign -f /root/keys/{key} -n maxor-release manifest.json
        """)

    def check(*flags):
        """maxor release check --json: (código de salida, estado)."""
        code, out = client.execute("maxor release check --json " + " ".join(flags) + " 2>/dev/null")
        return code, json.loads(out)

    def notes():
        code, out = client.execute("cat /tmp/notifications 2>/dev/null")
        return [l for l in out.splitlines() if l.strip()]

    with subtest("sin manifiesto el canal responde 404: no se dice 'al día'"):
        code, st = check("--force")
        assert code == 4, code
        assert st["status"] == "unavailable" and st["reason"] == "http_404", st
        assert st["available"] is False, st

    with subtest("una release nueva y bien firmada se ve como disponible"):
        publish("0.3.0", 1000)
        code, st = check("--force")
        assert code == 0, code
        assert st["status"] == "ok" and st["available"] is True and st["latest"] == "0.3.0", st

    with subtest("la segunda consulta usa la petición condicional (304) y sigue en ok"):
        client.succeed("maxor release check --force --quiet")
        server.succeed("grep -q ' 304 ' /var/log/nginx/access.log")
        code, st = check("--force")
        assert st["status"] == "ok" and st["available"] is True, st

    with subtest("avisa una sola vez por versión"):
        client.succeed("maxor release check --notify --quiet --force")
        client.succeed("maxor release check --notify --quiet --force")
        n = notes()
        assert len(n) == 1 and "0.3.0" in n[0], n

    with subtest("un manifiesto alterado en el servidor se rechaza y se avisa una vez"):
        server.succeed("sleep 1")
        server.succeed("sed -i 's/0.3.0/9.9.9/g' /srv/channel/manifest.json")
        code, st = check("--force")
        assert code == 1, code
        assert st["status"] == "insecure" and st["reason"] == "bad_signature" and st["available"] is False, st
        client.execute("maxor release check --notify --quiet --force")
        client.execute("maxor release check --notify --quiet --force")
        n = notes()
        assert len(n) == 2 and "critical" in n[1], n

    with subtest("firmado con otra clave no se acepta"):
        publish("0.3.0", 1000, key="evil")
        code, st = check("--force")
        assert st["status"] == "insecure" and st["reason"] == "bad_signature", st

    with subtest("un manifiesto sin firma se rechaza"):
        publish("0.3.0", 1000)
        server.succeed("rm /srv/channel/manifest.json.sig")
        code, st = check("--force")
        assert st["status"] == "insecure" and st["reason"] == "unsigned", st

    with subtest("sin servidor no se dice 'al día' y se conserva lo último verificado"):
        publish("0.3.0", 1000)
        code, st = check("--force")
        assert st["status"] == "ok", st
        server.systemctl("stop nginx.service")
        code, st = check("--force")
        assert code == 4, code
        assert st["status"] == "unavailable" and st["reason"] == "network", st
        assert st["available"] is True and st["ok_at"] > 0, st
        server.systemctl("start nginx.service")
        server.wait_for_unit("nginx.service")

    with subtest("una secuencia menor que la ya vista es un retroceso y se rechaza"):
        publish("0.2.5", 900)
        code, st = check("--force")
        assert st["status"] == "insecure" and st["reason"] == "rollback", st

    with subtest("el manifiesto del script de publicación de verdad lo acepta el cliente"):
        server.succeed("""
          cd /srv/channel
          ${releaseTools}/scripts/release-manifest.sh 0.4.0 cccccccccccccccccccccccccccccccccccccccc > manifest.json
          rm -f manifest.json.sig
          ssh-keygen -Y sign -f /root/keys/good -n maxor-release manifest.json
        """)
        code, st = check("--force")
        assert st["status"] == "ok" and st["commit"] == "c" * 40 and st["latest"] == "0.4.0" and st["available"] is True, st
  '';
}
