{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "grove-dev";
  runtimeInputs = lib'.envs.grove.dependencies;
  text = builtins.readFile ./main.sh;
}
