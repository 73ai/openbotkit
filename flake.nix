{
  description = "OpenBotKit";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" ];
      forAll = f: nixpkgs.lib.genAttrs systems (s: f nixpkgs.legacyPackages.${s});
      version = self.shortRev or self.dirtyShortRev or "dev";
    in
    {
      packages = forAll (pkgs: rec {
        obk = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) {
          pname = "obk";
          inherit version;
          src = self;
          vendorHash = "sha256-T9XL3Vlt7d1JxYnlZ0KSqQubpVIH58u52DrypM48b84=";
          subPackages = [ "." ];
          env.CGO_ENABLED = 0;
          ldflags = [ "-s" "-w" "-X github.com/73ai/openbotkit/internal/cli.Version=${version}" ];
          doCheck = false;
          postInstall = "ln -s $out/bin/openbotkit $out/bin/obk";
          meta.mainProgram = "obk";
        };
        default = obk;
      });

      devShells = forAll (pkgs: {
        default = pkgs.mkShell { packages = [ pkgs.go_1_26 pkgs.gopls ]; };
      });

      homeManagerModules.default = { config, lib, pkgs, osConfig ? null, ... }:
        let
          cfg = config.services.obk;
          home = config.home.homeDirectory;
          launchdFromDarwin = osConfig.services.obk.agentsFromHome or false;
          env = { HOME = home; PATH = "${cfg.package}/bin:/usr/bin:/bin"; };
          args = name: [ "${cfg.package}/bin/obk" "service" "run" name ] ++ cfg.${name}.extraArgs;
          mkService = name: lib.mkIf cfg.${name}.enable {
            systemd.user.services."obk-${name}" = lib.mkIf pkgs.stdenv.hostPlatform.isLinux {
              Unit = { Description = "OpenBotKit ${name}"; After = [ "network.target" ]; };
              Service = {
                ExecStart = lib.escapeShellArgs (args name);
                Environment = lib.mapAttrsToList (k: v: "${k}=${v}") env;
                Restart = "on-failure";
                RestartSec = 5;
              };
              Install.WantedBy = [ "default.target" ];
            };
            launchd.agents."obk-${name}" = lib.mkIf (pkgs.stdenv.hostPlatform.isDarwin && !launchdFromDarwin) {
              enable = true;
              config = {
                Label = "com.openbotkit.obk.${name}";
                ProgramArguments = args name;
                EnvironmentVariables = env;
                RunAtLoad = true;
                KeepAlive = true;
                StandardOutPath = "${home}/.obk/${name}.log";
                StandardErrorPath = "${home}/.obk/${name}.log";
              };
            };
          };
          serviceOpts = name: {
            enable = lib.mkEnableOption "obk ${name}";
            extraArgs = lib.mkOption { type = lib.types.listOf lib.types.str; default = [ ]; };
          };
        in
        {
          options.services.obk = {
            package = lib.mkOption {
              type = lib.types.package;
              default = self.packages.${pkgs.stdenv.hostPlatform.system}.obk;
            };
            daemon = serviceOpts "daemon";
            server = serviceOpts "server";
          };
          config = lib.mkMerge [
            (lib.mkIf (cfg.daemon.enable || cfg.server.enable) { home.packages = [ cfg.package ]; })
            (mkService "daemon")
            (mkService "server")
          ];
        };

      darwinModules.default = { config, lib, ... }:
        let
          agentsFor = user: hm:
            let
              cfg = hm.services.obk;
              home = hm.home.homeDirectory;
              agent = name: lib.optionalAttrs cfg.${name}.enable {
                "obk-${name}".serviceConfig = {
                  Label = "com.openbotkit.obk.${name}";
                  ProgramArguments = [ "${cfg.package}/bin/obk" "service" "run" name ] ++ cfg.${name}.extraArgs;
                  EnvironmentVariables = { HOME = home; PATH = "${cfg.package}/bin:/usr/bin:/bin"; };
                  RunAtLoad = true;
                  KeepAlive = true;
                  StandardOutPath = "${home}/.obk/${name}.log";
                  StandardErrorPath = "${home}/.obk/${name}.log";
                };
              };
            in
            lib.optionalAttrs (hm.services ? obk) (agent "daemon" // agent "server");
        in
        {
          options.services.obk.agentsFromHome = lib.mkEnableOption
            "launchd user agents for each home-manager user's obk services, run without a shell wrapper";

          config = lib.mkIf config.services.obk.agentsFromHome {
            launchd.user.agents = lib.mkMerge (lib.mapAttrsToList agentsFor (config.home-manager.users or { }));
          };
        };
    };
}
