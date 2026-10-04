# La instalación COMPLETA, de verdad: el motor corre todas sus etapas sobre un disco virtual vacío (comprobaciones,
# particionado, sistema de archivos, configuración de la máquina, nixos-install, arranque, cierre) y después se
# ARRANCA el sistema instalado desde ese disco. Sigue el patrón de los tests de instaladores de nixpkgs: dos máquinas,
# `installer` y `target`, que comparten el disco de destino. Se ejecuta con
#
#   nix build .#checks.x86_64-linux.installer-full -L
#
# Necesita KVM y tarda: evalúa y construye un sistema entero dentro de la VM. Sin red: todo lo que hace falta
# (las fuentes de los inputs del flake y el sistema casi entero) ya está en el store.
{ pkgs, self, inputs }:

let
  lib = pkgs.lib;
  maxor = pkgs.callPackage ../../packages/maxor.nix { };
  maxorInstall = pkgs.callPackage ../../packages/maxor-install.nix { inherit maxor; };

  # La contraseña de prueba es «maxor-test»; aquí solo viaja su hash.
  passwordHash = "$6$saltsalt12345678$wSY..HFnhYMY/6aLJYNhIOXVD5fW..6HYyu2ASnhxgQXHZXZPhPf/QgdVz2Jnj1FL5tZfWI2MekEMIQqUTz6j1";

  machine = {
    hostname = "testbox";
    user = "ana";
    fullname = "Ana Test";
    timezone = "Europe/Madrid";
    locale = "en_US.UTF-8";
    keymap = "es";
    xkb.layout = "es";
  };

  # What the installer takes from the machine, preseeded (the engine's MAXOR_INSTALL_HWJSON and _HWCONFIG): the
  # description of a typical Intel laptop, so the hardware's own packages are the ones a real machine would need.
  hwJson = ../../installer/generic/hardware-intel.json;
  # These are files of the repository, not derivations: the modules read them while evaluating, and a
  # derivation there would need building during evaluation (import from derivation), which CI forbids.
  hwConfig = ./installer-full/hardware-configuration.nix;
  localNix = ./installer-full/local.nix;

  # What the installation medium carries: a GENERIC Maxor OS. The machine installed here is not that one (another
  # hostname and account, another hardware description, the test harness's module): its fstab, initrd, units and top
  # level are built during the installation, offline, with the tools of installer/offline.nix. That is what this test
  # proves about the medium.
  generics = self.lib.offlineStore pkgs;

  # The test harness (the module that lets the test drive the installed system) is not part of Maxor OS and no real
  # machine has it: its packages are given to the installer as test infrastructure, in a system that is otherwise the
  # generic one, so that they are the only thing the test supplies that a real installation would not.
  harness = (self.lib.mkSystem {
    machine = { hostname = "maxor"; user = "maxor"; fullname = "Maxor"; };
    modules = [
      ../../installer/generic/hardware-configuration.nix
      ../../installer/generic/boot.nix
      localNix
      ({ ... }: {
        maxor.hardware.report = hwJson;
        maxor.settings = ../../installer/generic/maxor.json;
      })
    ];
  }).config.system.build.toplevel;

  # Sin red, el instalador usa copias locales de Maxor OS y de sus inputs (con sus metadatos) en vez de internet.
  overridesFile = pkgs.writeText "overrides" (import ../../installer/offline-overrides.nix { inherit lib self inputs; });

  commonConfig = {
    # el instalador construye cosas dentro de la VM: necesita memoria y espacio
    virtualisation.diskSize = 24 * 1024;
    virtualisation.cores = 4;
    virtualisation.memorySize = 8192;
    # el instalador y el sistema instalado usan el mismo disco (/dev/vda)
    virtualisation.diskImage = "./target.qcow2";
    virtualisation.useEFIBoot = true;
  };
in
pkgs.testers.runNixOSTest {
  name = "maxor-installer-full";
  # la instalación es larga
  globalTimeout = 5400;

  nodes = {
    installer = { config, pkgs, lib, ... }: {
      imports = [ commonConfig ../../installer/offline.nix ];

      # con el initrd de systemd, el dispositivo raíz vacío se formatea solo así
      virtualisation.fileSystems."/".autoFormat = true;

      # el sistema que corre el instalador vive en un disco pequeño aparte; /dev/vda queda libre para instalar
      virtualisation.emptyDiskImages = [ 1024 ];
      virtualisation.rootDevice = "/dev/vdb";
      # El sistema casi entero y las fuentes de los inputs (con su metadato, tal como las bloquea el flake.lock).
      virtualisation.additionalPaths = generics ++ [
        harness
        "${inputs.nixpkgs}"
        "${inputs.home-manager}"
        "${inputs.dms}"
        "${inputs.dms.inputs.dank-qml-common}"
        "${inputs.dms.inputs.flake-compat}"
      ];

      environment.systemPackages = [ maxorInstall pkgs.git pkgs.jq ];
      nix.settings = {
        experimental-features = [ "nix-command" "flakes" ];
        substituters = lib.mkForce [ ];
        hashed-mirrors = null;
        connect-timeout = 1;
      };

      # No network in the test: what is needed to evaluate and build a NixOS is what the installation medium carries.
    };

    # El sistema instalado, arrancado desde el mismo disco.
    target = { ... }: {
      imports = [ commonConfig ];
      virtualisation.useBootLoader = true;
      virtualisation.useDefaultFilesystems = false;
      virtualisation.efi.keepVariables = false;
      virtualisation.fileSystems."/" = {
        device = "/dev/disk/by-label/this-is-not-real-and-will-never-be-used";
        fsType = "ext4";
      };
    };
  };

  testScript = ''
    import json

    answers = {
        "schema": 1,
        "timezone": "${machine.timezone}",
        "locale": "${machine.locale}",
        "keymap": "${machine.keymap}",
        "xkb": {"layout": "${machine.xkb.layout}"},
        "network": {"offline": True},
        "disk": {"device": "/dev/vda", "strategy": "whole", "filesystem": "btrfs", "confirmed": "ERASE"},
        "machine": {"hostname": "${machine.hostname}"},
        "user": {"name": "${machine.user}", "fullname": "${machine.fullname}", "password_hash": "${passwordHash}"},
        "look": {"profiles": []},
    }

    installer.start()
    installer.wait_for_unit("multi-user.target")

    with subtest("the answers are valid and the plan is printable"):
        installer.succeed("cat > /tmp/a.json <<'EOF'\n" + json.dumps(answers) + "\nEOF")
        installer.succeed("maxor-install validate --answers /tmp/a.json")
        installer.succeed("maxor-install plan --answers /tmp/a.json > /tmp/plan.txt 2>&1")
        installer.succeed("grep -q 'nixos-install' /tmp/plan.txt")

    with subtest("a complete install with every stage of the engine"):
        installer.succeed(
            "MAXOR_INSTALL_MIN_GIB=8 "
            "MAXOR_INSTALL_EXTRA_ARGS=--impure "
            "MAXOR_INSTALL_LOCAL_NIX=${localNix} "
            "MAXOR_INSTALL_HWJSON=${hwJson} "
            "MAXOR_INSTALL_HWCONFIG=${hwConfig} "
            "MAXOR_INSTALL_OVERRIDES=\"$(cat ${overridesFile})\" "
            "maxor-install run --answers /tmp/a.json --events /tmp/events.jsonl >&2"
        )
        events = installer.succeed("cat /tmp/events.jsonl")
        states = [json.loads(l) for l in events.splitlines() if l.strip()]
        assert states[-1]["stage"] == "done" and states[-1]["state"] == "ok", states[-1]
        assert not any(s["state"] == "fail" for s in states), states
        installer.succeed("sync")

    installer.shutdown()

    # the same machine, booting the disk the installer wrote
    target.state_dir = installer.state_dir
    target.start()
    target.wait_for_unit("multi-user.target")

    with subtest("the installed system is Maxor OS for the account that was asked for"):
        assert target.succeed("hostname").strip() == "${machine.hostname}"
        target.succeed("id ${machine.user}")
        assert "Ana Test" in target.succeed("getent passwd ${machine.user}")
        assert target.succeed("timedatectl show -p Timezone --value").strip() == "${machine.timezone}"
        target.succeed("grep -q 'Maxor OS' /etc/os-release")

    with subtest("the password was set from the answers, outside the Nix store"):
        shadow = target.succeed("getent shadow ${machine.user}")
        assert "${passwordHash}" in shadow, shadow
        # Only the files the installer writes are checked: the repository's own sources (which hold this test) are
        # in the installed store too, and they name the hash because the test needs it.
        gen = "--include=flake.nix --include=local.nix --include=boot.nix --include=maxor.json --include=hardware.json --include=flake.lock"
        target.fail(f"grep -rl 'saltsalt12345678' /nix/store {gen} 2>/dev/null")
        target.fail("grep -rl 'saltsalt12345678' /home/${machine.user}/nixos-config /etc/nixos /var/log/maxor-install.log 2>/dev/null")

    with subtest("the machine configuration is the user's own git repository"):
        d = "/home/${machine.user}/nixos-config"
        target.succeed(f"test -f {d}/flake.nix -a -f {d}/host/local.nix -a -f {d}/flake.lock")
        target.succeed(f"test \"$(stat -c %U {d})\" = ${machine.user}")
        target.succeed(f"git -C {d} -c safe.directory='*' status --porcelain | wc -l | grep -qx 0")
        assert "maxor-os.lib.mkSystem" in target.succeed(f"cat {d}/flake.nix")

    with subtest("the CLI works and knows its trust keys"):
        # maxor lives in the user's profile (home-manager), so it is run as the user, in a login shell.
        out = target.succeed("su - ${machine.user} -c 'maxor version --json'")
        assert json.loads(out)["version"], out
        out = target.succeed("su - ${machine.user} -c 'maxor release status --json'")
        assert json.loads(out)["status"] in ("never", "ok", "unavailable"), out

    with subtest("boot entries and the log of the install are there"):
        target.succeed("test -e /boot/loader/loader.conf")
        target.succeed("test -s /var/log/maxor-install.log")
  '';
}
