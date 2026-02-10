{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "stone-lint";
  runtimeInputs = lib'.envs.stone.dependencies;
  text = builtins.readFile ./main.sh;
}
