{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "grove-build";
  runtimeInputs = lib'.envs.grove.dependencies;
  text = builtins.readFile ./main.sh;
}
