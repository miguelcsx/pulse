{ lib', pkgs }: {
  roots-db-start = pkgs.callPackage ./roots-db-start { inherit lib'; };
  roots-db-stop = pkgs.callPackage ./roots-db-stop { inherit lib'; };
  roots-db-create = pkgs.callPackage ./roots-db-create { inherit lib'; };
  roots-redis-start = pkgs.callPackage ./roots-redis-start { };
  roots-redis-stop = pkgs.callPackage ./roots-redis-stop { };
  roots-services = pkgs.callPackage ./roots-services { inherit lib'; };
  roots-sops = pkgs.callPackage ./roots-sops { };
}
