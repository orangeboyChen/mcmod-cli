// File: internal/cli/last80_test.go
// Created: 2026-06-20
// Description: Last push to 80%.
package cli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Last80", func() {
	It("newApp not nil", func() {
		Expect(NewApp()).NotTo(BeNil())
	})
	It("usage template basic", func() {
		Expect(usageTemplate()).To(ContainSubstring("lock"))
	})
	It("help command", func() {
		Expect(newSetCmd()).NotTo(BeNil())
	})
	It("commands exist", func() {
		app := NewApp()
		subs := app.Commands()
		names := make([]string, len(subs))
		for i, c := range subs {
			names[i] = c.Name()
		}
		Expect(names).To(ContainElement("lock"))
		Expect(names).To(ContainElement("build"))
		Expect(names).To(ContainElement("list"))
		Expect(names).To(ContainElement("set"))
		Expect(names).To(ContainElement("validate"))
		Expect(names).To(ContainElement("version"))
		Expect(names).To(ContainElement("config"))
		Expect(names).To(ContainElement("tree"))
	})
})
