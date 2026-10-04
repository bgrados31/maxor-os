# Las etapas REALES del instalador (comprobaciones previas, disco, LUKS y sistema de archivos) sobre discos
# virtuales, en una máquina que arrancó con UEFI. Es la capa que ningún simulacro puede probar: que sfdisk,
# cryptsetup, mkfs y mount hagan de verdad lo que el plan dice. Se ejecuta con
#
#   nix build .#checks.x86_64-linux.installer-disks -L
#
# Necesita KVM. La instalación completa (nixos-install y arrancar el resultado) es otra prueba.
{ pkgs }:

let
  maxor = pkgs.callPackage ../../packages/maxor.nix { };
  maxorInstall = pkgs.callPackage ../../packages/maxor-install.nix { inherit maxor; };
in
pkgs.testers.runNixOSTest {
  name = "maxor-installer-disks";

  nodes.machine = { pkgs, ... }: {
    virtualisation = {
      useEFIBoot = true;
      memorySize = 3072;
      emptyDiskImages = [ 4096 4096 4096 ]; # vdb, vdc, vdd
    };
    boot.supportedFilesystems = [ "btrfs" "ext4" "vfat" ];
    environment.systemPackages = [ maxorInstall ] ++ (with pkgs; [ btrfs-progs cryptsetup e2fsprogs dosfstools util-linux file jq ]);
  };

  testScript = ''
    import json

    ESP = "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
    ROOT = "4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709"
    ENV = "MAXOR_INSTALL_MIN_GIB=2 MAXOR_INSTALL_MIN_REGION_GIB=1"

    def write_answers(disk, **disk_extra):
        a = {
            "schema": 1,
            "network": {"offline": True},
            "disk": {"device": disk, "strategy": "whole", "confirmed": "ERASE", **disk_extra},
            "machine": {"hostname": "testbox"},
            "user": {"name": "ana", "password_hash": "$6$salt1234$abcdefghijklmnopqrstuvwxyz"},
        }
        machine.succeed("cat > /tmp/a.json <<'EOF'\n" + json.dumps(a) + "\nEOF")

    def install(stages, extra="", secret=None):
        pre = f"exec 7<<<'{secret}'; " if secret else ""
        post = " --secret-fd 7" if secret else ""
        return f"{pre}{ENV} maxor-install run --answers /tmp/a.json --reset --only {stages}{post} {extra}"

    def part_types(disk):
        return machine.succeed(f"sfdisk --json {disk}")

    machine.wait_for_unit("multi-user.target")

    with subtest("the machine booted with UEFI, so the real preflight can run"):
        machine.succeed("test -d /sys/firmware/efi")

    with subtest("preflight refuses the disk the system is running from, before touching anything"):
        before = machine.succeed("wipefs /dev/vda | sha256sum")  # lists signatures, read-only
        write_answers("/dev/vda")
        machine.fail(install("preflight,disk"))
        out = machine.execute(install("preflight,disk") + " 2>&1")[1]
        assert "is mounted" in out, out
        assert before == machine.succeed("wipefs /dev/vda | sha256sum"), "the running disk was modified"

    with subtest("a whole disk with btrfs: GPT with ESP and root, subvolumes and mounts"):
        write_answers("/dev/vdb", filesystem="btrfs")
        machine.succeed(install("preflight,disk,luks,filesystem"))
        table = json.loads(part_types("/dev/vdb"))["partitiontable"]
        parts = table["partitions"]
        assert table["label"] == "gpt", table
        assert [p["type"] for p in parts] == [ESP, ROOT], parts
        assert [p["name"] for p in parts] == ["MAXOR-ESP", "maxor-root"], parts
        assert parts[0]["size"] == 2097152, parts[0]  # 1 GiB
        assert parts[0]["start"] % 2048 == 0 and parts[1]["start"] % 2048 == 0  # aligned to 1 MiB
        assert machine.succeed("blkid -s LABEL -o value /dev/vdb1").strip() == "MAXOR-ESP"
        assert machine.succeed("blkid -s LABEL -o value /dev/vdb2").strip() == "maxor-root"
        assert machine.succeed("findmnt -no FSTYPE /mnt/boot").strip() == "vfat"
        assert machine.succeed("findmnt -no FSTYPE /mnt").strip() == "btrfs"
        assert "subvol=/@" in machine.succeed("findmnt -no OPTIONS /mnt")
        subs = machine.succeed("btrfs subvolume list /mnt")
        for name in ["@", "@home", "@snapshots"]:
            assert f"path {name}" in subs, subs
        assert "subvol=/@home" in machine.succeed("findmnt -no OPTIONS /mnt/home")
        assert "compress=zstd" in machine.succeed("findmnt -no OPTIONS /mnt")
        machine.succeed("umount -R /mnt")
        # The installed system mounts these by label, as its fstab does, in a fresh mount: it must work.
        machine.succeed("udevadm settle")
        machine.succeed("mkdir -p /t && mount -o subvol=@,compress=zstd,noatime /dev/disk/by-label/maxor-root /t")
        machine.succeed("ls -la /t >&2; test -d /t/home && test -d /t/.snapshots")
        machine.succeed("mount -o subvol=@home,compress=zstd,noatime /dev/disk/by-label/maxor-root /t/home")
        machine.succeed("mount -o subvol=@snapshots,compress=zstd,noatime /dev/disk/by-label/maxor-root /t/.snapshots")
        machine.succeed("umount -R /t")

    with subtest("ext4 with LUKS2 and a swap file: encrypted root, passphrase only on stdin"):
        write_answers("/dev/vdc", filesystem="ext4", encrypt={"enabled": True}, swap={"kind": "file", "gib": 1})
        machine.succeed(install("preflight,disk,luks,filesystem", secret="correct horse battery staple"))
        machine.succeed("cryptsetup isLuks /dev/vdc2")
        machine.succeed("cryptsetup luksDump /dev/vdc2 | grep -q 'Version:.*2'")
        machine.succeed("test -b /dev/mapper/maxor-root")
        assert machine.succeed("findmnt -no FSTYPE /mnt").strip() == "ext4"
        assert machine.succeed("findmnt -no SOURCE /mnt").strip().startswith("/dev/mapper/maxor-root")
        assert machine.succeed("stat -c %s /mnt/swapfile").strip() == str(1024 * 1024 * 1024)
        assert 'TYPE="swap"' in machine.succeed("blkid /mnt/swapfile")
        machine.succeed("echo -n 'correct horse battery staple' | cryptsetup open --test-passphrase /dev/vdc2")
        machine.fail("echo -n 'wrong passphrase here' | cryptsetup open --test-passphrase /dev/vdc2")
        machine.fail("grep -rq 'correct horse' /var/log/maxor-install.log /run/maxor-install")
        machine.succeed("umount -R /mnt; cryptsetup close maxor-root")

    with subtest("alongside another system: only the free region is used, what exists is untouched"):
        # a disk that already has "another system": an ESP and a data partition with a marker, 2 GiB free at the end
        machine.succeed("printf 'label: gpt\\nstart=2048, size=204800, type=U\\nstart=206848, size=2097152, type=EBD0A0A2-B9E5-4433-87C0-68B6B72699C7, name=Windows\\n' | sfdisk /dev/vdd")
        machine.succeed("mkfs.vfat /dev/vdd1; mkfs.ext4 -q -L Windows /dev/vdd2; mkdir -p /w && mount /dev/vdd2 /w && echo precious > /w/marker && umount /w")
        sums = machine.succeed("sha256sum /dev/vdd1 /dev/vdd2")

        probe = json.loads(machine.succeed("maxor-install probe"))
        disk = [d for d in probe if d["path"] == "/dev/vdd"][0]
        assert len(disk["partitions"]) == 2 and not disk["mounted"], disk
        gap = max(disk["free"], key=lambda g: g["sectors"])
        assert gap["sectors"] > 2 * 1024 * 2048, gap  # more than 2 GiB free

        def alongside(start, end, confirmed="INSTALL"):
            base = {"strategy": "alongside", "region": {"start": start, "end": end}, "filesystem": "ext4"}
            write_answers("/dev/vdd", **base)
            h = machine.succeed("maxor-install hash --answers /tmp/a.json").strip()
            write_answers("/dev/vdd", **base, plan_hash=h, confirmed=confirmed)

        # a region that overlaps the existing Windows partition is refused and nothing changes
        alongside(gap["start"] - 4096, gap["end"])
        machine.fail(install("preflight,disk,filesystem"))
        assert sums == machine.succeed("sha256sum /dev/vdd1 /dev/vdd2"), "existing partitions changed"
        assert len(json.loads(part_types("/dev/vdd"))["partitiontable"]["partitions"]) == 2

        # the real free region works
        alongside(gap["start"], gap["end"])
        machine.succeed(install("preflight,disk,filesystem"))
        parts = json.loads(part_types("/dev/vdd"))["partitiontable"]["partitions"]
        assert len(parts) == 4, parts
        assert [p["type"] for p in parts[2:]] == [ESP, ROOT], parts
        assert parts[2]["start"] >= gap["start"] and parts[3]["start"] + parts[3]["size"] - 1 <= gap["end"], parts
        assert sums == machine.succeed("sha256sum /dev/vdd1 /dev/vdd2"), "existing partitions changed"
        machine.succeed("umount -R /mnt")
        machine.succeed("mkdir -p /w && mount -o ro /dev/vdd2 /w && test \"$(cat /w/marker)\" = precious && umount /w")

    with subtest("a typo in the confirmation changes nothing"):
        write_answers("/dev/vdb", confirmed="erase")
        machine.fail(install("preflight,disk"))
  '';
}
