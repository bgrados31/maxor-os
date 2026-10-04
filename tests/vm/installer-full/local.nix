# The extension point host/local.nix of the installed machine. Here it makes the installed system talk to the
# test harness (a shell over the virtual console).
{ lib, modulesPath, ... }: {
  imports = [ (modulesPath + "/testing/test-instrumentation.nix") ];
  nix.settings.substituters = lib.mkForce [ ];
  # the harness needs to see the boot messages; Maxor's quiet boot hides them
  boot.consoleLogLevel = lib.mkForce 7;
}
