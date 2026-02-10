{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "grove-lint";
  runtimeInputs = lib'.envs.grove.dependencies;
  text = builtins.readFile ./main.sh;
}
