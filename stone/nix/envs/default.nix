{ inputs', pkgs, projectPath }: {
  stone = pkgs.callPackage ./stone { inherit inputs' projectPath; };
}
