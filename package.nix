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
    "-X github.com/gshpychka/fob/internal/store.agePath=${age}/bin/age"
  ]
  # age reaches a Mac's Secure Enclave through this plugin
  ++ lib.optionals stdenv.hostPlatform.isDarwin [
    "-X github.com/gshpychka/fob/internal/store.agePluginsPath=${lib.makeBinPath [ age-plugin-se ]}"
  ];

  # the store's test seals with age
  nativeCheckInputs = [ age ];

  # the tests serve stand-ins for the AWS endpoints on the loopback interface
  __darwinAllowLocalNetworking = true;

  meta = {
    description = "Sign-in to AWS IAM Identity Center and credentials for the roles it grants";
    mainProgram = "fob";
    platforms = lib.platforms.unix;
  };
}
