package pihole

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pi-Hole Client", func() {
	Context("When reconciling a resource", func() {
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			Expect(true).To(BeFalse())
		})
	})
})
