package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"

	"github.com/alphagov/paas-rubbernecker/pkg/pagerduty"
	"github.com/alphagov/paas-rubbernecker/pkg/pivotal"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

var _ = Describe("Main", func() {
	Context("providing correct environment variables have been set", func() {
		BeforeEach(func() {
			os.Setenv("PIVOTAL_TRACKER_API_TOKEN", "qwerty123456")
			os.Setenv("PIVOTAL_TRACKER_PROJECT_ID", "123456")
		})

		AfterEach(func() {
			os.Unsetenv("PIVOTAL_TRACKER_API_TOKEN")
			os.Unsetenv("PIVOTAL_TRACKER_PROJECT_ID")
		})

		It("should setupPivotal() successfully", func() {
			pr, err := setupPivotal()

			Expect(err).NotTo(HaveOccurred())
			Expect(pr).NotTo(BeNil())
		})
	})

	Context("provided environment variables have been set inappropriately", func() {
		var (
			err error
			pt  *pivotal.Tracker
		)

		It("should fail to setupPivotal() due to lack of project id", func() {
			pt, err = setupPivotal()

			Expect(err).To(HaveOccurred())
			Expect(pt).To(BeNil())
		})

		It("should fail to setupPivotal() due to lack of api token", func() {
			pt, err := setupPivotal()
			os.Setenv("PIVOTAL_TRACKER_PROJECT_ID", "123456")

			Expect(err).To(HaveOccurred())
			Expect(pt).To(BeNil())

			os.Unsetenv("PIVOTAL_TRACKER_PROJECT_ID")
		})
	})

	Context("provided everything has been setup correctly", func() {
		var (
			err error
			pt  *pivotal.Tracker
			pd  *pagerduty.Schedule

			apiURL          = `https://www.pivotaltracker.com/services/v5/projects/123456/stories?fields=owner_ids,blockers,transitions,current_state,labels,name,url,created_at&filter=state:started,finished,delivered,rejected`
			apiURLMembers   = `https://www.pivotaltracker.com/services/v5/projects/123456/memberships`
			apiURLSupport   = `https://api.pagerduty.com/oncalls`
			response        = `[{"blockers": [{"name":1234}],"transitions": [],"name": "Test Rubbernecker","current_state": "started","url": "http://localhost/story/show/561","owner_ids":[1234],"labels":[]}]`
			responseMembers = `[{"person":{"id":1234,"name":"Tester"}}]`
			responseSupport = `{"oncalls":[{"user":{"summary":"tester"},"schedule":{"summary":"test"}},{"user":{"summary":"tester"}}]}`
		)

		BeforeEach(func() {
			os.Setenv("PIVOTAL_TRACKER_API_TOKEN", "qwerty123456")
			os.Setenv("PIVOTAL_TRACKER_PROJECT_ID", "123456")

			pt, err = setupPivotal()

			Expect(err).NotTo(HaveOccurred())
			Expect(pt).NotTo(BeNil())

			pd = pagerduty.New("qwerty123456")

			httpmock.Activate()
		})

		AfterEach(func() {
			os.Unsetenv("PIVOTAL_TRACKER_API_TOKEN")
			os.Unsetenv("PIVOTAL_TRACKER_PROJECT_ID")

			httpmock.DeactivateAndReset()
		})

		It("should fail to fetchStories() due to non-responsive API", func() {
			httpmock.RegisterResponder("GET", apiURL,
				httpmock.NewStringResponder(500, ``))

			err = fetchStories(pt)

			Expect(err).To(HaveOccurred())
		})

		It("should fail to fetchStories() due to faulty API", func() {
			httpmock.RegisterResponder("GET", apiURL,
				httpmock.NewStringResponder(200, `[]`))

			err = fetchStories(pt)

			Expect(err).To(HaveOccurred())
		})

		It("should fail to fetchStories() due to lack of members", func() {
			httpmock.RegisterResponder("GET", apiURL,
				httpmock.NewStringResponder(200, response))

			err = fetchStories(pt)

			Expect(err).To(HaveOccurred())
		})

		It("should fail to fetchUsers() due to faulty API", func() {
			httpmock.RegisterResponder("GET", apiURLMembers,
				httpmock.NewStringResponder(200, `[]`))

			err = fetchUsers(pt)

			Expect(err).To(HaveOccurred())
		})

		It("should fetchUsers() successfully", func() {
			httpmock.RegisterResponder("GET", apiURLMembers,
				httpmock.NewStringResponder(200, responseMembers))

			err = fetchUsers(pt)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should fetchStories() successfully", func() {
			httpmock.RegisterResponder("GET", apiURL,
				httpmock.NewStringResponder(200, response))

			err = fetchStories(pt)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should fail to fetchSupport() due to faulty API", func() {
			httpmock.RegisterResponder("GET", apiURLSupport,
				httpmock.NewStringResponder(200, `[]`))

			err = fetchSupport(pd)

			Expect(err).To(HaveOccurred())
		})

		It("should fetchSupport() successfully", func() {
			httpmock.RegisterResponder("GET", apiURLSupport,
				httpmock.NewStringResponder(200, responseSupport))

			err = fetchSupport(pd)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should deal healthcheckHandler() correctly", func() {
			req, err := http.NewRequest("GET", "/health-check", nil)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(healthcheckHandler)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(ContainSubstring(`{"message":"OK"}`))
		})

		It("should deal indexHandler() correctly expecting Not Modified", func() {
			req, err := http.NewRequest("GET", "/", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("Accept", "application/json")
			req.Header.Add("If-None-Match", strconv.FormatInt(etag.Unix(), 10))

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(indexHandler)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusNotModified))
			Expect(rr.Body.String()).To(ContainSubstring(`{}`))
		})

		It("should deal indexHandler() correctly expecting JSON", func() {
			req, err := http.NewRequest("GET", "/", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("Accept", "application/json")

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(indexHandler)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
			Expect(rr.Body.String()).To(ContainSubstring(`"title":"Test Rubbernecker"`))
		})

		It("should deal indexHandler() correctly expecting HTML", func() {
			req, err := http.NewRequest("GET", "/", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Add("Accept", "text/html")

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(indexHandler)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Header().Get("Content-Type")).To(ContainSubstring("text/html"))
			Expect(rr.Body.String()).To(ContainSubstring(`<!doctype html>`))
		})

		It("should deal stylesHandler() correctly", func() {
			req, err := http.NewRequest("GET", "/style.css", nil)
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(stylesHandler)
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Header().Get("Content-Type")).To(ContainSubstring("text/css"))
			Expect(rr.Body.String()).To(ContainSubstring(`body{`))
		})
	})
})
