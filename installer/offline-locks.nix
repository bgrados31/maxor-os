# La referencia de internet de cada fuente que el medio lleva copiada, por su narHash (JSON: narHash → locked).
#
# Sin red, el flake.lock de la máquina se fija con copias locales (`path:/nix/store/…`, ver offline-overrides.nix).
# La etapa host lo reescribe después con estas referencias (github, con el mismo rev y narHash): Nix sigue tomando
# la copia local, porque la encuentra por su narHash, pero si un día la borra la limpieza del almacén, la vuelve a
# bajar de internet en vez de fallar. Y el lock queda igual que uno hecho con red.
{ lib, self }:

let
  lock = builtins.fromJSON (builtins.readFile (self + "/flake.lock"));
  fromLock = lib.filter (l: l ? narHash) (map (n: n.locked or { }) (lib.attrValues lock.nodes));
  # el propio Maxor OS: solo con un rev (un medio construido desde un commit, no desde un árbol con cambios)
  own = lib.optional (self ? rev) {
    type = "github";
    owner = "bgrados31";
    repo = "maxor-os";
    inherit (self) rev narHash lastModified;
  };
in
builtins.toJSON (lib.listToAttrs (map (l: lib.nameValuePair l.narHash l) (fromLock ++ own)))
