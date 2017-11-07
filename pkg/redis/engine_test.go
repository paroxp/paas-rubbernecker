package redis_test

import (
	"os"
	"strconv"

	rre "github.com/paroxp/paas-rubbernecker/pkg/redis"
	"github.com/go-redis/redis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Redis Engine", func() {
	var (
		re = rre.Engine{
			Client: redis.NewClient(&redis.Options{
				Addr:     os.Getenv("REDIS_URL"),
				Password: "",
				DB:       0,
			}),
		}
		arn      = "rubbernecker.pkg.redis.engine.test"
		arnValue = 123
	)

	It("should Put() value successfully", func() {
		err := re.Put(arn, arnValue)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should Get() value successfully", func() {
		value, err := re.Get(arn)

		Expect(err).NotTo(HaveOccurred())
		Expect(value).To(Equal(strconv.Itoa(arnValue)))
	})

	It("should fail to Get() value", func() {
		value, err := re.Get("arn")

		Expect(err).To(HaveOccurred())
		Expect(value).To(BeEmpty())
	})
})
