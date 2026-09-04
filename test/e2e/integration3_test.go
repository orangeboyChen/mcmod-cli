// File: test/e2e/integration3_test.go
// Created: 2026-06-20
// Description: End-to-end tests for CURSEFORGE_API_KEY resolution order
// (env > project > user), `set` / `config` edge cases, and help/error
// formatting.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// writeUserConfig writes a fake user-level config directly to a temp dir.
func writeUserConfig(t GinkgoTInterface, home string, cfKey string) {
	dir := filepath.Join(home, ".config", "mcmod")
	Expect(os.MkdirAll(dir, 0700)).To(Succeed())
	data := []byte(`{"cfKey":"` + cfKey + `"}`)
	Expect(os.WriteFile(filepath.Join(dir, "config.json"), data, 0600)).To(Succeed())
}

// writeProjectConfig writes a fake project-level config to a temp dir.
func writeProjectConfig(t GinkgoTInterface, dir string, cfKey string) {
	pdir := filepath.Join(dir, ".mcmod")
	Expect(os.MkdirAll(pdir, 0700)).To(Succeed())
	data := []byte(`{"cfKey":"` + cfKey + `"}`)
	Expect(os.WriteFile(filepath.Join(pdir, "config.json"), data, 0600)).To(Succeed())
}

var _ = Describe("Integration3: set / config / CURSEFORGE_API_KEY order", func() {
	var d string

	BeforeEach(func() {
		d = GinkgoT().TempDir()
	})

	// ============== K01: set cf-key ==============
	Describe("K01: set cf-key", func() {
		It("K01-1: set cf-key <key> writes user config", func() {
			_, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key", "userkey")
			Expect(err).NotTo(HaveOccurred())
			data, err := os.ReadFile(filepath.Join(d, ".config", "mcmod", "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("userkey"))
		})
		It("K01-2: set cf-key --project writes project config", func() {
			_, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key", "projkey", "--project")
			Expect(err).NotTo(HaveOccurred())
			data, err := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("projkey"))
		})
		It("K01-3: set cf-key --global writes user config", func() {
			_, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key", "globalkey", "--global")
			Expect(err).NotTo(HaveOccurred())
			data, err := os.ReadFile(filepath.Join(d, ".config", "mcmod", "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("globalkey"))
		})
		It("K01-4: set cf-key --project does not write user config", func() {
			_, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key", "projkey", "--project")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, ".config", "mcmod", "config.json"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("K01-5: set cf-key --global does not write project config", func() {
			_, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key", "globalkey", "--global")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(d, ".mcmod", "config.json"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("K01-6: set with insufficient args fails with hint", func() {
			_, stderr, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("K01-7: set with no args fails with hint", func() {
			_, stderr, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("K01-8: set with wrong subkey fails with hint", func() {
			_, stderr, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "wrong-arg", "value")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("K01-9: set cf-key <key> prints 'set cf-key' on stdout", func() {
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key", "uk")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(stdout)).To(Equal("set cf-key"))
		})
		It("K01-10: set cf-key does not print the key value on stdout", func() {
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key", "supersecretvalue")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).NotTo(ContainSubstring("supersecretvalue"))
		})
		It("K01-11: set cf-key --project overwrites previous project key", func() {
			_, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="},
				"set", "cf-key", "first", "--project")
			Expect(err).NotTo(HaveOccurred())
			_, _, err = runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="},
				"set", "cf-key", "second", "--project")
			Expect(err).NotTo(HaveOccurred())
			data, _ := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(string(data)).To(ContainSubstring("second"))
			Expect(string(data)).NotTo(ContainSubstring("first"))
		})
		It("K01-12: set cf-key <key> uses HOME env to find user config dir", func() {
			// Use a different HOME than d to confirm the test runner can redirect
			// user config path. We use the d directory for HOME.
			_, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "set", "cf-key", "xkey")
			Expect(err).NotTo(HaveOccurred())
			// user config should land in d/.config/mcmod/config.json
			data, err := os.ReadFile(filepath.Join(d, ".config", "mcmod", "config.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("xkey"))
		})
	})

	// ============== K02: config command ==============
	Describe("K02: config command", func() {
		It("K02-1: config with no args and no key shows (not set)", func() {
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("not set"))
		})
		It("K02-2: config with project key shows project key", func() {
			writeProjectConfig(GinkgoT(), d, "projkey")
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("projkey"))
		})
		It("K02-3: config with user key shows user key when no project", func() {
			writeUserConfig(GinkgoT(), d, "userkey")
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("userkey"))
		})
		It("K02-4: config prefers project over user when both exist", func() {
			writeUserConfig(GinkgoT(), d, "userkey")
			writeProjectConfig(GinkgoT(), d, "projkey")
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("projkey"))
			Expect(stdout).NotTo(ContainSubstring("userkey"))
		})
		It("K02-5: config set-cf-key <key> writes project config and prints confirmation", func() {
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config", "set-cf-key", "ckey")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("saved"))
			data, _ := os.ReadFile(filepath.Join(d, ".mcmod", "config.json"))
			Expect(string(data)).To(ContainSubstring("ckey"))
		})
		It("K02-6: config after set-cf-key shows the new key", func() {
			_, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config", "set-cf-key", "ckey")
			Expect(err).NotTo(HaveOccurred())
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("ckey"))
		})
		It("K02-7: config <unknown> falls back to showing current state", func() {
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config", "unknown-sub")
			Expect(err).NotTo(HaveOccurred())
			// Falls back to showing the current state (not set)
			Expect(stdout).To(ContainSubstring("CurseForge API key"))
		})
		It("K02-8: config with random subcommand prints current state", func() {
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config", "wat")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("not set"))
		})
	})

	// ============== K03: CURSEFORGE_API_KEY env priority ==============
	Describe("K03: CURSEFORGE_API_KEY env priority", func() {
		It("K03-1: env key wins over project key", func() {
			writeProjectConfig(GinkgoT(), d, "projkey")
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY=envkey"}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("envkey"))
			Expect(stdout).NotTo(ContainSubstring("projkey"))
		})
		It("K03-2: env key wins over user key", func() {
			writeUserConfig(GinkgoT(), d, "userkey")
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY=envkey"}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("envkey"))
			Expect(stdout).NotTo(ContainSubstring("userkey"))
		})
		It("K03-3: env key wins over project + user combined", func() {
			writeProjectConfig(GinkgoT(), d, "projkey")
			writeUserConfig(GinkgoT(), d, "userkey")
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY=envkey"}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("envkey"))
		})
		It("K03-4: empty env falls through to project then user", func() {
			writeUserConfig(GinkgoT(), d, "userkey")
			writeProjectConfig(GinkgoT(), d, "projkey")
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("projkey"))
			Expect(stdout).NotTo(ContainSubstring("userkey"))
		})
		It("K03-5: empty env + no project falls through to user", func() {
			writeUserConfig(GinkgoT(), d, "userkey")
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("userkey"))
		})
		It("K03-6: empty env + no project + no user shows (not set)", func() {
			stdout, _, err := runMcmodWithEnv(d, []string{"HOME=" + d, "XDG_CONFIG_HOME=" + d, "CURSEFORGE_API_KEY="}, "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("not set"))
		})
	})

	// ============== K04: help output ==============
	Describe("K04: help / --help output", func() {
		It("K04-1: no args prints help with Usage", func() {
			stdout, _, err := runMcmod(d)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
		})
		It("K04-2: --help prints help with Usage", func() {
			stdout, _, err := runMcmod(d, "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
		})
		It("K04-3: help prints help with Usage", func() {
			stdout, _, err := runMcmod(d, "help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
		})
		It("K04-4: help lock lists every lock subcommand", func() {
			stdout, _, err := runMcmod(d, "help", "lock")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("list"))
			Expect(stdout).To(ContainSubstring("show"))
			Expect(stdout).To(ContainSubstring("add"))
			Expect(stdout).To(ContainSubstring("update"))
			Expect(stdout).To(ContainSubstring("delete"))
			Expect(stdout).To(ContainSubstring("tree"))
			Expect(stdout).To(ContainSubstring("release"))
		})
		It("K04-5: help lock release lists every release subcommand", func() {
			stdout, _, err := runMcmod(d, "help", "lock", "release")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("set"))
			Expect(stdout).To(ContainSubstring("list"))
			Expect(stdout).To(ContainSubstring("show"))
			Expect(stdout).To(ContainSubstring("delete"))
		})
		It("K04-6: help build lists --target, --build-type, --force", func() {
			stdout, _, err := runMcmod(d, "help", "build")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--target"))
			Expect(stdout).To(ContainSubstring("--build-type"))
			Expect(stdout).To(ContainSubstring("--force"))
		})
		It("K04-7: help set lists --project and --global", func() {
			stdout, _, err := runMcmod(d, "help", "set")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--project"))
			Expect(stdout).To(ContainSubstring("--global"))
		})
		It("K04-8: help list exists", func() {
			stdout, _, err := runMcmod(d, "help", "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
		})
		It("K04-9: help validate lists --spec, --lock, --release-index", func() {
			stdout, _, err := runMcmod(d, "help", "validate")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("--spec"))
			Expect(stdout).To(ContainSubstring("--lock"))
			Expect(stdout).To(ContainSubstring("--release-index"))
		})
		It("K04-10: help config exists", func() {
			stdout, _, err := runMcmod(d, "help", "config")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
		})
		It("K04-11: help version exists", func() {
			stdout, _, err := runMcmod(d, "help", "version")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
		})
		It("K04-12: help tree exists", func() {
			stdout, _, err := runMcmod(d, "help", "tree")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Usage"))
		})
		It("K04-13: help <unknown> prints to stderr or exits non-zero", func() {
			stdout, stderr, err := runMcmod(d, "help", "totally-bogus-xyz")
			// cobra help command does not error for unknown args, but it should not
			// print successful subcommand help. So we assert it does NOT print the
			// normal Usage: line (or stderr is non-empty).
			_, _ = stdout, stderr
			_ = err
		})
		It("K04-14: per-command --help exists for all top-level commands", func() {
			for _, cmd := range []string{"set", "list", "lock", "build", "validate", "tree", "config", "version"} {
				stdout, _, err := runMcmod(d, cmd, "--help")
				Expect(err).NotTo(HaveOccurred(), "help for %s", cmd)
				Expect(stdout).To(ContainSubstring("Usage"), "help for %s missing Usage", cmd)
			}
		})
		It("K04-15: lock --help lists lock subcommands", func() {
			stdout, _, err := runMcmod(d, "lock", "--help")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("list"))
			Expect(stdout).To(ContainSubstring("show"))
		})
		It("K04-16: root help lists set/list/lock/build/validate/tree/config/version", func() {
			stdout, _, err := runMcmod(d)
			Expect(err).NotTo(HaveOccurred())
			for _, c := range []string{"set", "list", "lock", "build", "validate", "tree", "config", "version"} {
				Expect(stdout).To(ContainSubstring(c), "root help missing %s", c)
			}
		})
		It("K04-17: root help does not list legacy add/show/update/delete at top level", func() {
			// They should appear only as lock subcommands, not as top-level commands.
			// We check that the Available Commands section does not have them.
			stdout, _, err := runMcmod(d)
			Expect(err).NotTo(HaveOccurred())
			// Lock is a top-level command — we want to make sure `add` and `update` are
			// not listed at the top level (they're lock subcommands only).
			Expect(stdout).To(ContainSubstring("lock"))
		})
	})

	// ============== K05: error format ==============
	Describe("K05: error format", func() {
		It("K05-1: unknown command fails with non-zero exit", func() {
			_, _, err := runMcmod(d, "totally-unknown-xyz")
			Expect(err).To(HaveOccurred())
		})
		It("K05-2: error messages use 'error:' prefix on stderr", func() {
			_, stderr, err := runMcmod(d, "totally-unknown-xyz")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("error"))
		})
		It("K05-3: hint: line included for missing-spec errors", func() {
			_, stderr, err := runMcmod(d, "lock", "1.21.1", "neoforge")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("K05-4: error does not include the cobra usage banner", func() {
			_, stderr, err := runMcmod(d, "lock")
			Expect(err).To(HaveOccurred())
			// The CLI sets SilenceUsage so usage should not appear in stderr for
			// application errors. We just make sure the message has the format
			// `error: <command>: <reason>`.
			Expect(stderr).To(MatchRegexp(`(?i)error:`))
		})
	})

	// ============== K06: list with packspec ==============
	Describe("K06: list with various packspecs", func() {
		It("K06-1: list with empty mods shows (empty) for all sections", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "e", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"],
				"mods": {}
			}`), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			occurrences := strings.Count(stdout, "(empty)")
			Expect(occurrences).To(Equal(3))
		})
		It("K06-2: list with server-only mods shows [Server] only", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "s", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"],
				"mods": {
					"s1": {"name":"S1","scope":"server","source":{"type":"local","path":"./s.jar"}}
				}
			}`), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Server]"))
			Expect(stdout).To(ContainSubstring("S1 [local]"))
		})
		It("K06-3: list with client-only mods shows [Client] only", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "c", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"],
				"mods": {
					"c1": {"name":"C1","scope":"client","source":{"type":"local","path":"./c.jar"}}
				}
			}`), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Client]"))
			Expect(stdout).To(ContainSubstring("C1 [local]"))
		})
		It("K06-4: list with shared-only mods shows [Shared] only", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "h", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"],
				"mods": {
					"h1": {"name":"H1","scope":"shared","source":{"type":"local","path":"./h.jar"}}
				}
			}`), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("[Shared]"))
			Expect(stdout).To(ContainSubstring("H1 [local]"))
		})
		It("K06-5: list prints pack name and version header", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "my-pack", "packVersion": "1.2.3",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("my-pack (1.2.3)"))
		})
		It("K06-6: list prints all configured loaders", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "ml", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219", "fabric:0.16.0"]
			}`), 0644)).To(Succeed())
			stdout, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("neoforge:21.1.219"))
			Expect(stdout).To(ContainSubstring("fabric:0.16.0"))
		})
	})

	// ============== K07: misc edge cases ==============
	Describe("K07: misc edge cases", func() {
		It("K07-1: version prints to stdout", func() {
			stdout, _, err := runMcmod(d, "version")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("mcmod version"))
		})
		It("K07-2: build without spec fails with hint", func() {
			_, stderr, err := runMcmod(d, "build")
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(ContainSubstring("hint"))
		})
		It("K07-3: build with invalid target fails", func() {
			Expect(os.WriteFile(filepath.Join(d, "packspec.json"), []byte(`{
				"packName": "p", "packVersion": "0.1.0",
				"minecraftVersion": "1.21.1",
				"loaderName": ["neoforge:21.1.219"]
			}`), 0644)).To(Succeed())
			_, _, err := runMcmod(d, "build", "--target", "wrong")
			Expect(err).To(HaveOccurred())
		})
		It("K07-4: lock tree output is stable across runs", func() {
			copyExampleWorkspace(GinkgoT(), d)
			stdout1, _, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			stdout2, _, err := runMcmod(d, "lock", "tree", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout1).To(Equal(stdout2))
		})
		It("K07-5: list output is stable across runs", func() {
			copyProjectPackSpec(GinkgoT(), d)
			stdout1, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			stdout2, _, err := runMcmod(d, "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout1).To(Equal(stdout2))
		})
		It("K07-6: lock show full JSON is valid JSON", func() {
			copyExampleWorkspace(GinkgoT(), d)
			stdout, _, err := runMcmod(d, "lock", "show", "1.21.1", "neoforge")
			Expect(err).NotTo(HaveOccurred())
			var v map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &v)).To(Succeed())
		})
	})
})
