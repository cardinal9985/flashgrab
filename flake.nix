{
  description = "flashgrab - download Flash and browser games from the web";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            goreleaser
            vhs
          ];
        };

        packages.default = pkgs.buildGoModule {
          pname = "flashgrab";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-9kKzk+hD1TKePc0sOPyJgG66PQGRaHZ72KltJvVUYAo=";
        };
      });
}
