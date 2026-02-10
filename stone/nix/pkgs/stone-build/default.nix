{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "stone-build";
  runtimeInputs = lib'.envs.stone.dependencies;
  text = builtins.readFile ./main.sh;
}
