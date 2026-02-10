{
  description = "Pulse — ember (React PWA)";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-25.05";

    roots = {
      inputs = {
        flake-parts.follows = "flake-parts";
        nixpkgs.follows = "nixpkgs";
      };
      url = "path:../roots";
    };
  };

  outputs = inputs:
    inputs.flake-parts.lib.mkFlake { inherit inputs; } {
      debug = false;

      systems = [ "aarch64-darwin" "aarch64-linux" "x86_64-linux" ];

      perSystem = { inputs', pkgs, self', system, ... }:
        let
          projectPath = path: ./. + path;
          lib' = {
            envs = import ./nix/envs { inherit pkgs; };
            inherit projectPath;
          };
        in {
          _module.args.pkgs = import inputs.nixpkgs {
            inherit system;
            config.allowUnfree = true;
          };

          devShells =
            import ./nix/shells.nix { inherit inputs' lib' pkgs self'; };
          packages = import ./nix/pkgs { inherit lib' pkgs; };
        };
    };
}
