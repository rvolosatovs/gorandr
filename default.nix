with import <nixpkgs>{};

buildGoPackage rec {
  name = "gorandr";

  goPackagePath = "github.com/rvolosatovs/gorandr";
  subPackages = [ "cmd/randrq" ];

  src = ./.;

  goDeps = ./deps.nix;
  
  meta = with stdenv.lib; {
    description = "X11 RandR helper";
    license = licenses.mit;
    homepage = https://github.com/rvolosatovs/gorandr;
  };
}
