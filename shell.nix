let 
  pkgs = import /home/tom/rdpkgs {};
  src = pkgs.fetchgit {
    fetchSubmodules = true;
    url = "https://github.com/notedit/media-server-go-native.git";
    deepClone = true;
    rev = "4b84572c5b39f08e4dfaa44f0fe9e0c0f6b73e63";
    sha256 = "sha256-+vrlixx8Z5RlKeafvBER7CClmMPNdEag3EFoeyWgTrs=";
  };
in
with pkgs;

let
  openssl =
pkgs.stdenv.mkDerivation  {
  nativeBuildInputs = [
    autoconf automake libtool
    perl
  ];
  name = "media-server-go-native-openssl";
  inherit src;
  buildPhase = ''
    cd ./openssl &&  export KERNEL_BITS=64 && ./config --prefix=$out --openssldir=$out && make && make install
  '';
};
  srtp =
pkgs.stdenv.mkDerivation  {
  nativeBuildInputs = [
    autoconf automake libtool
    perl openssl
  ];
  name = "media-server-go-native-srtp";
  inherit src;
  buildPhase = ''
    cd ./libsrtp && ./configure --prefix=$out --enable-openssl  --with-openssl-dir=${openssl}  && make && make install
  '';
};
  mp4v2 =
pkgs.stdenv.mkDerivation  {
  nativeBuildInputs = [
    autoconf automake libtool
    perl openssl srtp
  ];
  name = "media-server-go-native-mp4v2";
  inherit src;
  buildPhase = ''
    cd ./mp4v2 && autoreconf -i
    ./configure --prefix=$out
   make
  '';
};
  media-server-go-native =
pkgs.stdenv.mkDerivation  {
  nativeBuildInputs = [
    autoconf automake libtool
    perl openssl srtp mp4v2
  ];
  name = "media-server-go-native";
  inherit src;
  buildPhase = ''
    cp media-Makefile  ./media-server/Makefile
    export ROOT_DIR=$PWD
    export CXXFLAGS="$CXXFLAGS -msse4.2 -I$src/media-server/ext/crc32c/config/Linux-x86_64"

    cp config.mk  ./media-server/
    cd media-server
    make libmediaserver.a
    mkdir -p $out/lib
    cp ./bin/release/libmediaserver.a  $out/lib/
  '';
  installPhase = " ";
};

in mkShell rec {
  inputsFrom = buildInputs;
  buildInputs = [
    openssl
    srtp
    mp4v2
    media-server-go-native

    gst_all_1.gstreamer
    gst_all_1.gst-plugins-base
    gst_all_1.gst-plugins-good
    gst_all_1.gst-plugins-bad
    gst_all_1.gst-plugins-ugly
    gst_all_1.gst-libav
    gst_all_1.gst-vaapi

    gst_all_1.gstreamer.dev
    gst_all_1.gst-plugins-bad.dev

  ];
  shellHook = ''
    export CGO_CXXFLAGS="-I${mp4v2}/include -I${media-server-go-native}/include"
    export CGO_LDFLAGS="-L${mp4v2}/lib -L${media-server-go-native}/lib $LD_LIBRARY_PATH -lmp4v2 -lmediaserver"

    # For Gstreamer plugins
    export GST_PLUGIN_PATH=$GST_PLUGIN_PATH:$PWD
    export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$PWD
    echo Setting GST_PLUGIN_PATH to $PWD
    echo Adding LD_LIBRARY_PATH to $PWD


  '';
}
