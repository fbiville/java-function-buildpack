/*
 * Copyright 2018 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package java_test

import (
	"fmt"
	"github.com/projectriff/streaming-http-adapter-buildpack/adapter"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/openjdk-cnb/jre"
	. "github.com/onsi/gomega"
	"github.com/projectriff/java-function-buildpack/java"
	"github.com/projectriff/libfnbuildpack/function"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestRiffInvoker(t *testing.T) {
	spec.Run(t, "RiffJavaInvoker", func(t *testing.T, when spec.G, it spec.S) {

		g := NewGomegaWithT(t)

		when("Detect", func() {

			var f *test.DetectFactory

			it.Before(func() {
				f = test.NewDetectFactory(t)
			})

			it("contains openjdk-jre and riff-invoker-java in build plan", func() {
				g.Expect(java.BuildPlanContribution(f.Detect, function.Metadata{Handler: "myfunction"})).To(Equal(buildplan.BuildPlan{
					java.Dependency: buildplan.Dependency{
						Metadata: buildplan.Metadata{java.Handler: "myfunction"},
					},
					jre.Dependency: buildplan.Dependency{
						Metadata: buildplan.Metadata{jre.LaunchContribution: true},
					},
					adapter.Dependency: buildplan.Dependency{
						Metadata: buildplan.Metadata{},
					},
				}))
			})
		})

		when("Build", func() {
			var f *test.BuildFactory

			it.Before(func() {
				f = test.NewBuildFactory(t)
				f.AddDependency(java.Dependency, filepath.Join("testdata", "stub-invoker.jar"))
			})

			it("returns true if build plan exists", func() {
				f.AddBuildPlan(java.Dependency, buildplan.Dependency{
					Metadata: buildplan.Metadata{java.Handler: "myfunction"},
				})

				_, ok, err := java.NewJavaInvoker(f.Build)
				g.Expect(ok).To(BeTrue())
				g.Expect(err).NotTo(HaveOccurred())
			})

			it("returns false if build plan does not exist", func() {
				_, ok, err := java.NewJavaInvoker(f.Build)
				g.Expect(ok).To(BeFalse())
				g.Expect(err).NotTo(HaveOccurred())
			})

			it("contributes invoker", func() {
				f.AddBuildPlan(java.Dependency, buildplan.Dependency{
					Metadata: buildplan.Metadata{java.Handler: "myfunction"},
				})

				r, _, err := java.NewJavaInvoker(f.Build)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(r.Contribute()).To(Succeed())

				layer := f.Build.Layers.Layer("riff-invoker-java")
				g.Expect(layer).To(test.HaveLayerMetadata(false, false, true))
				g.Expect(filepath.Join(layer.Root, "META-INF", "MANIFEST.MF")).To(BeARegularFile())

				command := fmt.Sprintf("streaming-http-adapter java -cp %s $JAVA_OPTS org.springframework.boot.loader.JarLauncher",
					layer.Root)

				g.Expect(f.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
					Processes: layers.Processes{
						layers.Process{Type: "function", Command: command},
						layers.Process{Type: "web", Command: command},
					},
				}))

				functionLayer := f.Build.Layers.Layer("function")
				g.Expect(functionLayer).To(test.HaveOverrideLaunchEnvironment("SPRING_CLOUD_FUNCTION_DEFINITION", "myfunction"))
				g.Expect(functionLayer).To(test.HaveOverrideLaunchEnvironment("SPRING_CLOUD_FUNCTION_LOCATION", f.Build.Application.Root))
			})
		})
	}, spec.Report(report.Terminal{}))
}
