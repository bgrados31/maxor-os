# La lista «entrada=referencia» que permite instalar sin red (MAXOR_INSTALL_OVERRIDES): cada entrada del flake de
# la máquina apunta a su copia local en el disco, en vez de a internet. La usan la ISO y las pruebas.
#
# Importa conservar los metadatos de cada fuente (rev, lastModified, narHash): NixOS deriva de ellos su cadena de
# versión y, si faltan, cambian las derivaciones de medio sistema y todo se tendría que compilar de nuevo.
# Por eso la referencia es `path:/nix/store/…?rev=…&lastModified=…&narHash=…` y no un `path:` a secas.
{ lib, self, inputs }:

let
  ref = i:
    let
      params = lib.filter (p: p != "") [
        (lib.optionalString (i ? rev) "rev=${i.rev}")
        (lib.optionalString (i ? lastModified) "lastModified=${toString i.lastModified}")
        (lib.optionalString (i ? narHash) "narHash=${lib.escapeURL i.narHash}")
      ];
    in
    "path:${i}${lib.optionalString (params != [ ]) "?${lib.concatStringsSep "&" params}"}";
in
''
  maxor-os=path:${self}
  maxor-os/nixpkgs=${ref inputs.nixpkgs}
  maxor-os/home-manager=${ref inputs.home-manager}
  maxor-os/dms=${ref inputs.dms}
  maxor-os/dms/dank-qml-common=${ref inputs.dms.inputs.dank-qml-common}
  maxor-os/dms/flake-compat=${ref inputs.dms.inputs.flake-compat}
''
