{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "stone-test";
  runtimeInputs = lib'.envs.stone.dependencies;
  text = builtins.readFile ./main.sh;
}
