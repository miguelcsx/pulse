{ inputs', lib', pkgs }:
pkgs.writeShellApplication {
  name = "stone-local";
  runtimeInputs = pkgs.lib.flatten [
    lib'.envs.stone.dependencies

    (pkgs.lib.attrValues inputs'.roots.packages)

    pkgs.bash
    pkgs.mprocs
  ];
  text = builtins.readFile ./main.sh;
}
