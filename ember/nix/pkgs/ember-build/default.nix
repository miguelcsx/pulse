{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "ember-build";
  runtimeInputs = lib'.envs.ember.dependencies;
  text = builtins.readFile ./main.sh;
}
