{
  lib,
  stdenv,
  buildGoModule,
  age,
  age-plugin-se,
  fzf,
}:
buildGoModule {
  pname = "fob";
  version = "0.1.0";

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      ./go.mod
      (lib.fileset.fileFilter (file: file.hasExt "go") ./.)
    ];
  };

  # the standard library is the only dependency
  vendorHash = null;
  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-w"
    "-X main.fzfPath=${lib.getExe fzf}"
  ]
  # the Secure Enclave store exists only on macOS
  ++ lib.optionals stdenv.hostPlatform.isDarwin [
    "-X github.com/gshpychka/fob/internal/store.agePath=${age}/bin/age"
    "-X github.com/gshpychka/fob/internal/store.agePluginSEPath=${age-plugin-se}/bin/age-plugin-se"
  ];

  # the tests serve stand-ins for the AWS endpoints on the loopback interface
  __darwinAllowLocalNetworking = true;

  meta = {
    description = "Sign-in to AWS IAM Identity Center and credentials for the roles it grants";
    mainProgram = "fob";
    platforms = lib.platforms.unix;
  };
}
