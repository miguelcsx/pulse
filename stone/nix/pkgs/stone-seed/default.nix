{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "stone-seed";
  runtimeInputs = pkgs.lib.flatten [
    lib'.envs.stone.dependencies
    lib'.envs.stone.envars
  ];
  text = builtins.readFile ./main.sh;
}
