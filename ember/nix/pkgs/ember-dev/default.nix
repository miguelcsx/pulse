{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "ember-dev";
  runtimeInputs = lib'.envs.ember.dependencies;
  text = builtins.readFile ./main.sh;
}
