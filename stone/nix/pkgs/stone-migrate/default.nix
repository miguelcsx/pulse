{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "stone-migrate";
  runtimeInputs = pkgs.lib.flatten [
    lib'.envs.stone.dependencies
    lib'.envs.stone.envars
  ];
  text = builtins.readFile ./main.sh;
}
