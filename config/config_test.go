package config_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpack/pack/config"
	h "github.com/buildpack/pack/testhelpers"
)

func TestConfig(t *testing.T) {
	color.NoColor = true
	spec.Run(t, "config", testConfig, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testConfig(t *testing.T, when spec.G, it spec.S) {
	var tmpDir string

	it.Before(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "pack.config.test.")
		h.AssertNil(t, err)
	})

	it.After(func() {
		err := os.RemoveAll(tmpDir)
		h.AssertNil(t, err)
	})

	when("#New", func() {
		when("no config on disk", func() {
			it("writes the defaults to disk", func() {
				subject, err := config.New(tmpDir)
				h.AssertNil(t, err)

				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), `default-stack-id = "io.buildpacks.stacks.bionic"`)
				h.AssertContains(t, string(b), `default-builder = "packs/samples:v3alpha2"`)
				h.AssertContains(t, string(b), strings.TrimSpace(`
[[stacks]]
  id = "io.buildpacks.stacks.bionic"
  build-image = "packs/build:v3alpha2"
  run-images = ["packs/run:v3alpha2"]
`))

				h.AssertEq(t, len(subject.Stacks), 1)
				h.AssertEq(t, subject.Stacks[0].ID, "io.buildpacks.stacks.bionic")
				h.AssertEq(t, subject.Stacks[0].BuildImage, "packs/build:v3alpha2")
				h.AssertEq(t, len(subject.Stacks[0].RunImages), 1)
				h.AssertEq(t, subject.Stacks[0].RunImages[0], "packs/run:v3alpha2")
				h.AssertEq(t, subject.DefaultStackID, "io.buildpacks.stacks.bionic")
				h.AssertEq(t, subject.DefaultBuilder, "packs/samples:v3alpha2")
			})

			when("path is missing", func() {
				it("creates the directory", func() {
					_, err := config.New(filepath.Join(tmpDir, "a", "b"))
					h.AssertNil(t, err)

					b, err := ioutil.ReadFile(filepath.Join(tmpDir, "a", "b", "config.toml"))
					h.AssertNil(t, err)
					h.AssertContains(t, string(b), `default-stack-id = "io.buildpacks.stacks.bionic"`)
				})
			})
		})

		when("config on disk is missing one of the built-in stacks", func() {
			it.Before(func() {
				w, err := os.Create(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				defer w.Close()
				w.Write([]byte(`
default-stack-id = "some.user.provided.stack"
default-builder = "some/builder"

[[stacks]]
  id = "some.user.provided.stack"
  build-image = "some/build"
  run-images = ["some/run"]
`))
			})

			it("add built-in stack while preserving custom stack, custom default-stack-id, and custom default-builder", func() {
				subject, err := config.New(tmpDir)
				h.AssertNil(t, err)

				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), `default-stack-id = "some.user.provided.stack"`)
				h.AssertContains(t, string(b), `default-builder = "some/builder"`)
				h.AssertContains(t, string(b), strings.TrimSpace(`
[[stacks]]
  id = "io.buildpacks.stacks.bionic"
  build-image = "packs/build:v3alpha2"
  run-images = ["packs/run:v3alpha2"]
`))
				h.AssertContains(t, string(b), strings.TrimSpace(`
[[stacks]]
  id = "some.user.provided.stack"
  build-image = "some/build"
  run-images = ["some/run"]
`))
				h.AssertEq(t, subject.DefaultStackID, "some.user.provided.stack")
				h.AssertEq(t, subject.DefaultBuilder, "some/builder")

				h.AssertEq(t, len(subject.Stacks), 2)
				h.AssertEq(t, subject.Stacks[0].ID, "some.user.provided.stack")
				h.AssertEq(t, subject.Stacks[0].BuildImage, "some/build")
				h.AssertEq(t, len(subject.Stacks[0].RunImages), 1)
				h.AssertEq(t, subject.Stacks[0].RunImages[0], "some/run")

				h.AssertEq(t, subject.Stacks[1].ID, "io.buildpacks.stacks.bionic")
				h.AssertEq(t, subject.Stacks[1].BuildImage, "packs/build:v3alpha2")
				h.AssertEq(t, len(subject.Stacks[1].RunImages), 1)
				h.AssertEq(t, subject.Stacks[1].RunImages[0], "packs/run:v3alpha2")
			})
		})

		when("config.toml already has the built-in stack", func() {
			it.Before(func() {
				w, err := os.Create(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				defer w.Close()
				w.Write([]byte(`
[[stacks]]
  id = "io.buildpacks.stacks.bionic"
  build-image = "some-other/build"
  run-images = ["some-other/run", "packs/run:v3alpha2"]
`))
			})

			it("does not modify the built-in stack if it is customized", func() {
				subject, err := config.New(tmpDir)
				h.AssertNil(t, err)

				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), `default-stack-id = "io.buildpacks.stacks.bionic"`)
				h.AssertContains(t, string(b), strings.TrimSpace(`
[[stacks]]
  id = "io.buildpacks.stacks.bionic"
  build-image = "some-other/build"
  run-images = ["some-other/run", "packs/run:v3alpha2"]
`))

				h.AssertEq(t, len(subject.Stacks), 1)
				h.AssertEq(t, subject.Stacks[0].ID, "io.buildpacks.stacks.bionic")
				h.AssertEq(t, subject.Stacks[0].BuildImage, "some-other/build")
				h.AssertEq(t, len(subject.Stacks[0].RunImages), 2)
				h.AssertEq(t, subject.Stacks[0].RunImages[0], "some-other/run")
				h.AssertEq(t, subject.Stacks[0].RunImages[1], "packs/run:v3alpha2")
				h.AssertEq(t, subject.DefaultStackID, "io.buildpacks.stacks.bionic")
			})
		})

		when("config.toml has an outdated built-in stack", func() {
			it.Before(func() {
				w, err := os.Create(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				defer w.Close()
				w.Write([]byte(`
default-builder = "packs/samples"

[[stacks]]
  id = "io.buildpacks.stacks.bionic"
  build-image = "packs/build"
  run-images = ["some-other/run", "packs/run"]
`))
			})

			it("modifies old defaults", func() {
				subject, err := config.New(tmpDir)
				h.AssertNil(t, err)

				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), `default-stack-id = "io.buildpacks.stacks.bionic"`)
				h.AssertContains(t, string(b), `default-builder = "packs/samples:v3alpha2"`)
				h.AssertContains(t, string(b), strings.TrimSpace(`
[[stacks]]
  id = "io.buildpacks.stacks.bionic"
  build-image = "packs/build:v3alpha2"
  run-images = ["some-other/run", "packs/run:v3alpha2"]
`))

				h.AssertEq(t, len(subject.Stacks), 1)
				h.AssertEq(t, subject.Stacks[0].ID, "io.buildpacks.stacks.bionic")
				h.AssertEq(t, subject.Stacks[0].BuildImage, "packs/build:v3alpha2")
				h.AssertEq(t, len(subject.Stacks[0].RunImages), 2)
				h.AssertEq(t, subject.Stacks[0].RunImages[0], "some-other/run")
				h.AssertEq(t, subject.Stacks[0].RunImages[1], "packs/run:v3alpha2")
				h.AssertEq(t, subject.DefaultStackID, "io.buildpacks.stacks.bionic")
				h.AssertEq(t, subject.DefaultBuilder, "packs/samples:v3alpha2")
			})
		})

		when("config.toml has an outdated format", func() {
			it.Before(func() {
				w, err := os.Create(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				defer w.Close()
				w.Write([]byte(`
default-builder = "packs/samples"

[[stacks]]
  id = "my.stack"
  build-images = ["some-other/build"]
  run-images = ["some-other/run"]
`))
			})

			it("modifies old defaults", func() {
				subject, err := config.New(tmpDir)
				h.AssertNil(t, err)

				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), strings.TrimSpace(`
[[stacks]]
  id = "my.stack"
  build-image = "some-other/build"
  run-images = ["some-other/run"]
`))

				h.AssertEq(t, subject.Stacks[0].ID, "my.stack")
				h.AssertEq(t, subject.Stacks[0].BuildImage, "some-other/build")
				h.AssertEq(t, len(subject.Stacks[0].BuildImages), 0)
			})
		})
	})

	when("Config#GetStack", func() {
		var subject *config.Config
		it.Before(func() {
			h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
default-stack-id = "my.stack"
[[stacks]]
  id = "stack-1"
[[stacks]]
  id = "my.stack"
[[stacks]]
  id = "stack-3"
`), 0666))
			var err error
			subject, err = config.New(tmpDir)
			h.AssertNil(t, err)
		})
		when("no stack is requested", func() {
			it("returns the default stack", func() {
				stack, err := subject.GetStack("")
				h.AssertNil(t, err)
				h.AssertEq(t, stack.ID, "my.stack")
			})
		})
		when("a stack known is requested", func() {
			it("returns the stack", func() {
				stack, err := subject.GetStack("stack-1")
				h.AssertNil(t, err)
				h.AssertEq(t, stack.ID, "stack-1")

				stack, err = subject.GetStack("stack-3")
				h.AssertNil(t, err)
				h.AssertEq(t, stack.ID, "stack-3")
			})
		})
		when("an unknown stack is requested", func() {
			it("returns an error", func() {
				_, err := subject.GetStack("stack-4")
				h.AssertNotNil(t, err)
				h.AssertEq(t, err.Error(), "stack 'stack-4' does not exist")
			})
		})
	})

	when("Config#SetDefaultStack", func() {
		var subject *config.Config
		it.Before(func() {
			h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
default-stack-id = "old.default.stack"
[[stacks]]
  id = "some.stack"
`), 0666))
			var err error
			subject, err = config.New(tmpDir)
			h.AssertNil(t, err)
		})

		when("the stack exists", func() {
			it("sets the default-stack-id", func() {
				err := subject.SetDefaultStack("some.stack")
				h.AssertNil(t, err)
				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), `default-stack-id = "some.stack"`)
			})
		})

		when("the stack doesn't exist", func() {
			it("returns an error and leaves the original default", func() {
				err := subject.SetDefaultStack("some.missing.stack")
				h.AssertError(t, err, "stack 'some.missing.stack' does not exist")
				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), `default-stack-id = "old.default.stack"`)
			})
		})
	})

	when("Config#SetDefaultBuilder", func() {
		var subject *config.Config
		it.Before(func() {
			h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
default-stack-id = "old/builder"
`), 0666))
			var err error
			subject, err = config.New(tmpDir)
			h.AssertNil(t, err)
		})

		it("sets the default-builder", func() {
			err := subject.SetDefaultBuilder("new/builder")
			h.AssertNil(t, err)
			b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
			h.AssertNil(t, err)
			h.AssertContains(t, string(b), `default-builder = "new/builder"`)
			h.AssertEq(t, subject.DefaultBuilder, "new/builder")
		})
	})

	when("Config#AddStack", func() {
		var subject *config.Config
		it.Before(func() {
			h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
default-stack-id = "my.stack"
[[stacks]]
  id = "stack-1"
[[stacks]]
  id = "my.stack"
[[stacks]]
  id = "stack-3"
`), 0666))
			var err error
			subject, err = config.New(tmpDir)
			h.AssertNil(t, err)
		})

		when("stack to be added is new", func() {
			it("adds the stack and writes to file", func() {
				err := subject.AddStack(config.Stack{
					ID:         "new-stack",
					BuildImage: "neworg/build",
					RunImages:  []string{"neworg/run"},
				})
				h.AssertNil(t, err)

				stack, err := subject.GetStack("new-stack")
				h.AssertNil(t, err)
				h.AssertEq(t, stack.ID, "new-stack")
				h.AssertEq(t, stack.BuildImage, "neworg/build")
				h.AssertEq(t, stack.RunImages, []string{"neworg/run"})

				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertContains(t, string(b), "new-stack")
				h.AssertContains(t, string(b), "neworg/build")
				h.AssertContains(t, string(b), "neworg/run")
			})
		})

		when("stack to be added is already in file", func() {
			it("errors and leaves file unchanged", func() {
				stat, err := os.Stat(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				origSize := stat.Size()

				err = subject.AddStack(config.Stack{
					ID:         "my.stack",
					BuildImage: "neworg/build",
					RunImages:  []string{"neworg/run"},
				})
				h.AssertNotNil(t, err)
				h.AssertEq(t, err.Error(), "stack 'my.stack' already exists")

				stack, err := subject.GetStack("my.stack")
				h.AssertNil(t, err)
				h.AssertEq(t, stack.BuildImage, "")

				stat, err = os.Stat(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				h.AssertEq(t, stat.Size(), origSize)
			})
		})
	})

	when("Config#UpdateStack", func() {
		var subject *config.Config
		it.Before(func() {
			h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
default-stack-id = "stack-1"
[[stacks]]
  id = "stack-1"
[[stacks]]
  id = "my.stack"
	build-image = "packs/build:v3alpha2"
	run-images = ["packs/run:v3alpha2"]
[[stacks]]
  id = "stack-3"
`), 0666))
			var err error
			subject, err = config.New(tmpDir)
			h.AssertNil(t, err)
		})

		when("stack to be updated exists", func() {
			it("updates the stack and writes the file", func() {
				err := subject.UpdateStack("my.stack", config.Stack{
					BuildImage: "packs/build-2",
					RunImages:  []string{"packs/run-2", "jane"},
				})
				h.AssertNil(t, err)

				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				if !strings.Contains(string(b), "packs/build-2") {
					t.Fatalf(`expected "packs/build-2" to be in config.toml: %s`, b)
				}
				if !strings.Contains(string(b), "packs/run-2") {
					t.Fatalf(`expected "packs/run-2" to be in config.toml: %s`, b)
				}
			})

			it("updates only the fields entered", func() {
				err := subject.UpdateStack("my.stack", config.Stack{
					BuildImage: "packs/build-2",
				})
				h.AssertNil(t, err)
				stack, err := subject.GetStack("my.stack")
				h.AssertNil(t, err)
				h.AssertEq(t, stack.BuildImage, "packs/build-2")
				h.AssertEq(t, stack.RunImages, []string{"packs/run:v3alpha2"})

				err = subject.UpdateStack("my.stack", config.Stack{
					RunImages: []string{"packs/run-3"},
				})
				h.AssertNil(t, err)
				stack, err = subject.GetStack("my.stack")
				h.AssertNil(t, err)
				h.AssertEq(t, stack.BuildImage, "packs/build-2")
				h.AssertEq(t, stack.RunImages, []string{"packs/run-3"})
			})
		})

		when("stack to be updated is NOT in file", func() {
			it("errors and leaves file unchanged", func() {
				err := subject.UpdateStack("other.stack", config.Stack{
					BuildImage: "packs/build-2",
					RunImages:  []string{"packs/run-2"},
				})
				h.AssertNotNil(t, err)
				h.AssertEq(t, err.Error(), "stack 'other.stack' does not exist")
			})
		})

		when("neither build image nor run image specified", func() {
			it("errors and leaves file unchanged", func() {
				err := subject.UpdateStack("my.stack", config.Stack{})
				h.AssertNotNil(t, err)
				h.AssertEq(t, err.Error(), "no build image or run image(s) specified")
			})
		})
	})

	when("Config#DeleteStack", func() {
		var subject *config.Config
		it.Before(func() {
			h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
default-stack-id = "stack-1"
[[stacks]]
  id = "stack-1"
[[stacks]]
  id = "my.stack"
[[stacks]]
  id = "stack-3"
`), 0666))
			var err error
			subject, err = config.New(tmpDir)
			h.AssertNil(t, err)
		})

		when("stack to be deleted exists", func() {
			it("deletes the stack and writes the file", func() {
				_, err := subject.GetStack("my.stack")
				h.AssertNil(t, err)

				err = subject.DeleteStack("my.stack")
				h.AssertNil(t, err)

				_, err = subject.GetStack("my.stack")
				h.AssertNotNil(t, err)

				b, err := ioutil.ReadFile(filepath.Join(tmpDir, "config.toml"))
				h.AssertNil(t, err)
				if strings.Contains(string(b), "my.stack") {
					t.Fatal(`expected "my.stack" to longer be in config.toml`)
				}
			})
		})

		when("stack to be deleted is NOT in file", func() {
			it("errors and leaves file unchanged", func() {
				err := subject.DeleteStack("other.stack")
				h.AssertNotNil(t, err)
				h.AssertEq(t, err.Error(), "stack 'other.stack' does not exist")
			})
		})

		when("stack to be deleted is the default-stack-id", func() {
			it("errors and leaves file unchanged", func() {
				err := subject.DeleteStack("stack-1")
				h.AssertNotNil(t, err)
				h.AssertEq(t, err.Error(), `stack-1 cannot be deleted when it is the default stack. You can change your default stack by running "pack set-default-stack".`)
			})
		})
	})

	when("Config#GetRunImage", func() {
		var subject *config.Config

		when("run image exists in config", func() {
			it.Before(func() {
				h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
[[run-images]]
  tag = "some/run-image"
  mirrors = ["some/run-image1", "some.registry/some/run-image"]
`), 0666))
				var err error
				subject, err = config.New(tmpDir)
				h.AssertNil(t, err)
			})

			it("returns the builder config", func() {
				runImage := subject.GetRunImage("some/run-image")
				h.AssertNotNil(t, runImage)
				h.AssertEq(t, runImage.Image, "some/run-image")
				h.AssertEq(t, len(runImage.Mirrors), 2)
				h.AssertSliceContains(t, runImage.Mirrors, "some/run-image1")
				h.AssertSliceContains(t, runImage.Mirrors, "some.registry/some/run-image")
			})
		})

		when("run image does not exist in config", func() {
			it.Before(func() {
				h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
[[run-images]]
  tag = "some-other/run-image"
  mirrors = ["some/run", "some.registry/some/run"]
`), 0666))
				var err error
				subject, err = config.New(tmpDir)
				h.AssertNil(t, err)
			})

			it("returns a nil pointer", func() {
				builder := subject.GetRunImage("some/builder")
				h.AssertNil(t, builder)
			})
		})
	})

	when("Config#SetRunImageMirrors", func() {
		var subject *config.Config

		when("run image exists in config", func() {
			it.Before(func() {
				h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(`
[[run-images]]
  tag = "some/run-image"
  mirrors = ["some/run", "some.registry/some/run"]
`), 0666))
				var err error
				subject, err = config.New(tmpDir)
				h.AssertNil(t, err)
			})

			it("updates the run image", func() {
				subject.SetRunImageMirrors("some/run-image", []string{"some-other/run"})

				reloadedConfig, err := config.New(tmpDir)
				h.AssertNil(t, err)

				image := reloadedConfig.GetRunImage("some/run-image")

				h.AssertNotNil(t, image)
				h.AssertEq(t, image.Image, "some/run-image")
				h.AssertEq(t, len(image.Mirrors), 1)
				h.AssertSliceContains(t, image.Mirrors, "some-other/run")
			})
		})

		when("run image does not exist in config", func() {
			it.Before(func() {
				h.AssertNil(t, ioutil.WriteFile(filepath.Join(tmpDir, "config.toml"), nil, 0666))
				var err error
				subject, err = config.New(tmpDir)
				h.AssertNil(t, err)
			})

			it("adds the run image", func() {
				subject.SetRunImageMirrors("some/run-image", []string{"some-other/run"})

				reloadedConfig, err := config.New(tmpDir)
				h.AssertNil(t, err)

				image := reloadedConfig.GetRunImage("some/run-image")

				h.AssertNotNil(t, image)
				h.AssertEq(t, image.Image, "some/run-image")
				h.AssertEq(t, len(image.Mirrors), 1)
				h.AssertSliceContains(t, image.Mirrors, "some-other/run")
			})
		})
	})

	when("ImageByRegistry", func() {
		var images []string
		it.Before(func() {
			images = []string{
				"first.com/org/repo",
				"myorg/myrepo",
				"zonal.gcr.io/org/repo",
				"gcr.io/org/repo",
			}
		})
		when("repoName is dockerhub", func() {
			it("returns the dockerhub image", func() {
				name, err := config.ImageByRegistry("index.docker.io", images)
				h.AssertNil(t, err)
				h.AssertEq(t, name, "myorg/myrepo")
			})
		})
		when("registry is gcr.io", func() {
			it("returns the gcr.io image", func() {
				name, err := config.ImageByRegistry("gcr.io", images)
				h.AssertNil(t, err)
				h.AssertEq(t, name, "gcr.io/org/repo")
			})
			when("registry is zonal.gcr.io", func() {
				it("returns the gcr image", func() {
					name, err := config.ImageByRegistry("zonal.gcr.io", images)
					h.AssertNil(t, err)
					h.AssertEq(t, name, "zonal.gcr.io/org/repo")
				})
			})
			when("registry is missingzone.gcr.io", func() {
				it("returns first run image", func() {
					name, err := config.ImageByRegistry("missingzone.gcr.io", images)
					h.AssertNil(t, err)
					h.AssertEq(t, name, "first.com/org/repo")
				})
			})
		})

		when("one of the images is non-parsable", func() {
			it.Before(func() {
				images = []string{"as@ohd@as@op", "gcr.io/myorg/myrepo"}
			})
			it("skips over it", func() {
				name, err := config.ImageByRegistry("gcr.io", images)
				h.AssertNil(t, err)
				h.AssertEq(t, name, "gcr.io/myorg/myrepo")
			})
		})
	})
}
