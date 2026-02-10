{ lib', pkgs }:
pkgs.writeShellApplication {
  name = "ember-lint";
  runtimeInputs = lib'.envs.ember.dependencies;
  text = builtins.readFile ./main.sh;
}
