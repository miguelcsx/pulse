{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "stone-seed";
  runtimeInputs = pkgs.lib.flatten [
    lib'.envs.stone.dependencies
  ];
  text = builtins.readFile ./main.sh;
}
