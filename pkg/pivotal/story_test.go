package pivotal_test

import (
	httpmock "gopkg.in/jarcoal/httpmock.v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/paroxp/paas-rubbernecker/pkg/pivotal"
	"github.com/paroxp/paas-rubbernecker/pkg/rubbernecker"
)

var _ = Describe("Pivotal Stories", func() {
	Context("Tracker not setup", func() {
		It("should create a New() tracker", func() {
			pt, err := pivotal.New(123, "test")

			Expect(err).NotTo(HaveOccurred())
			Expect(pt).NotTo(BeNil())
		})
	})

	Context("Tracker setup", func() {
		var (
			pt *pivotal.Tracker

			apiURL   = `https://www.pivotaltracker.com/services/v5/projects/123/stories?fields=owner_ids,blockers,transitions,current_state,labels,name,url,created_at&filter=state:started`
			response = `[{"blockers": [{"name":1234}],"transitions": [],"name": "Test Rubbernecker","current_state": "started","url": "http://localhost/story/show/561","owner_ids":[1234],"labels":[]}]`
		)

		BeforeEach(func() {
			var err error

			pt, err = pivotal.New(123, "test")
			httpmock.Activate()

			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			httpmock.DeactivateAndReset()
		})

		It("should fail to FetchCards() stories from an API", func() {
			httpmock.RegisterResponder("GET", apiURL,
				httpmock.NewStringResponder(404, ``))

			err := pt.FetchCards(rubbernecker.StatusDoing)

			Expect(err).To(HaveOccurred())
		})

		It("should FetchCards() stories from an API", func() {
			httpmock.RegisterResponder("GET", apiURL,
				httpmock.NewStringResponder(200, response))

			err := pt.FetchCards(rubbernecker.StatusDoing)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail to FlattenStories() due to faulty API", func() {
			httpmock.RegisterResponder("GET", apiURL,
				httpmock.NewStringResponder(200, `[]`))

			err := pt.FetchCards(rubbernecker.StatusDoing)

			Expect(err).NotTo(HaveOccurred())

			cards, err := pt.FlattenStories()

			Expect(err).To(HaveOccurred())
			Expect(cards).To(BeNil())
		})

		It("should FlattenStories() correctly", func() {
			httpmock.RegisterResponder("GET", apiURL,
				httpmock.NewStringResponder(200, response))

			err := pt.FetchCards(rubbernecker.StatusDoing)

			Expect(err).NotTo(HaveOccurred())

			cards, err := pt.FlattenStories()
			c := *cards

			Expect(err).NotTo(HaveOccurred())
			Expect(len(c)).To(Equal(1))
			Expect(c[0].Title).To(Equal("Test Rubbernecker"))
			Expect(len(c[0].Stickers)).To(Equal(1))
			Expect(c[0].Stickers[0]).To(Equal("blocked"))
		})
	})
})
