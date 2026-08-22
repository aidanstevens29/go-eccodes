package eccodes

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type valueProbe struct {
	Index int     `json:"index"`
	Value float64 `json:"value"`
}

type goldenGRIB struct {
	SHA256         string       `json:"sha256"`
	SourceSHA256   string       `json:"sourceSHA256"`
	VerifiedWith   string       `json:"verifiedWithEccodes"`
	MessageCount   int          `json:"messageCount"`
	Edition        int64        `json:"edition"`
	ShortName      string       `json:"shortName"`
	TypeOfLevel    string       `json:"typeOfLevel"`
	Level          int64        `json:"level"`
	DataDate       int64        `json:"dataDate"`
	DataTime       int64        `json:"dataTime"`
	StepRange      string       `json:"stepRange"`
	GridType       string       `json:"gridType"`
	Nx             int64        `json:"nx"`
	Ny             int64        `json:"ny"`
	NumberOfPoints int64        `json:"numberOfPoints"`
	BitsPerValue   int64        `json:"bitsPerValue"`
	PackingType    string       `json:"packingType"`
	TotalLength    int64        `json:"totalLength"`
	Minimum        float64      `json:"minimum"`
	Maximum        float64      `json:"maximum"`
	Average        float64      `json:"average"`
	ValueProbes    []valueProbe `json:"valueProbes"`
}

var _ = Describe("ecCodes", func() {
	Describe("version reporting", func() {
		It("reports the linked ecCodes version", func() {
			version := RuntimeVersion()

			Expect(version.Major).To(BeNumerically(">=", 2))
			Expect(version.String()).To(MatchRegexp(`^\d+\.\d+\.\d+$`))
		})
	})

	Describe("opening files", func() {
		It("rejects paths containing NUL bytes", func() {
			reader, err := Open("forecast\x00ignored.grib2")

			Expect(err).To(MatchError(ContainSubstring("path contains NUL byte")))
			Expect(reader).To(BeNil())
		})

		It("returns an error for a missing file", func() {
			path := filepath.Join(GinkgoT().TempDir(), "missing.grib2")

			reader, err := Open(path)

			Expect(err).To(HaveOccurred())
			Expect(reader).To(BeNil())
		})
	})

	Describe("in-memory messages", func() {
		It("rejects empty input", func() {
			message, err := NewMessage(nil)

			Expect(err).To(MatchError("eccodes: parse message: empty input"))
			Expect(message).To(BeNil())
		})
	})

	Describe("reader lifecycle", func() {
		It("allows repeated closes and rejects subsequent reads", func() {
			reader := &Reader{closed: true}

			Expect(reader.Close()).To(Succeed())
			Expect(reader.Close()).To(Succeed())
			message, err := reader.Next()
			Expect(message).To(BeNil())
			Expect(err).To(MatchError(ErrClosed))
		})
	})

	Describe("the NBM golden fixture", Label("integration", "golden", "nbm"), func() {
		It("matches the native ecCodes results", func() {
			fixturePath := filepath.Join("testdata", "nbm.t00z.core.f001.co.aptmp.grib2")
			expected := loadGolden(filepath.Join("testdata", "nbm.t00z.core.f001.co.aptmp.expected.json"))
			fixture, err := os.ReadFile(fixturePath)
			Expect(err).NotTo(HaveOccurred())
			checksum := sha256.Sum256(fixture)
			Expect(fmt.Sprintf("%x", checksum)).To(Equal(expected.SHA256))

			reader, err := Open(fixturePath)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() { Expect(reader.Close()).To(Succeed()) })

			message, err := reader.Next()
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() { Expect(message.Close()).To(Succeed()) })

			expectLong(message, "edition", expected.Edition)
			expectString(message, "shortName", expected.ShortName)
			expectString(message, "typeOfLevel", expected.TypeOfLevel)
			expectLong(message, "level", expected.Level)
			expectLong(message, "dataDate", expected.DataDate)
			expectLong(message, "dataTime", expected.DataTime)
			expectString(message, "stepRange", expected.StepRange)
			expectString(message, "gridType", expected.GridType)
			expectLong(message, "Nx", expected.Nx)
			expectLong(message, "Ny", expected.Ny)
			expectLong(message, "numberOfPoints", expected.NumberOfPoints)
			expectLong(message, "bitsPerValue", expected.BitsPerValue)
			expectString(message, "packingType", expected.PackingType)
			expectLong(message, "totalLength", expected.TotalLength)

			values, err := message.Doubles("values")
			Expect(err).NotTo(HaveOccurred())
			Expect(values).To(HaveLen(int(expected.NumberOfPoints)))
			for _, probe := range expected.ValueProbes {
				Expect(probe.Index).To(BeNumerically("<", len(values)))
				Expect(values[probe.Index]).To(
					BeNumerically("~", probe.Value, 1e-9),
					"decoded value at index %d",
					probe.Index,
				)
			}

			minimum, maximum, average := statistics(values)
			Expect(minimum).To(BeNumerically("~", expected.Minimum, 1e-9))
			Expect(maximum).To(BeNumerically("~", expected.Maximum, 1e-9))
			Expect(average).To(BeNumerically("~", expected.Average, 1e-9))
			expectDouble(message, "min", expected.Minimum)
			expectDouble(message, "max", expected.Maximum)
			expectDouble(message, "average", expected.Average)

			encoded, err := message.Bytes()
			Expect(err).NotTo(HaveOccurred())
			Expect(encoded).To(Equal(fixture))

			copy, err := NewMessage(encoded)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() { Expect(copy.Close()).To(Succeed()) })
			expectLong(copy, "edition", expected.Edition)
			expectString(copy, "shortName", expected.ShortName)
			copyValues, err := copy.Doubles("values")
			Expect(err).NotTo(HaveOccurred())
			Expect(copyValues).To(Equal(values))

			messageCount := 1
			for {
				next, err := reader.Next()
				if errors.Is(err, io.EOF) {
					break
				}
				Expect(err).NotTo(HaveOccurred())
				Expect(next.Close()).To(Succeed())
				messageCount++
			}
			Expect(messageCount).To(Equal(expected.MessageCount))
		})
	})
})

func loadGolden(path string) goldenGRIB {
	data, err := os.ReadFile(path)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	var expected goldenGRIB
	ExpectWithOffset(1, json.Unmarshal(data, &expected)).To(Succeed())
	return expected
}

func expectLong(message *Message, key string, expected int64) {
	value, err := message.Long(key)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, value).To(Equal(expected), "key %s", key)
}

func expectDouble(message *Message, key string, expected float64) {
	value, err := message.Double(key)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, value).To(BeNumerically("~", expected, 1e-9), "key %s", key)
}

func expectString(message *Message, key, expected string) {
	value, err := message.String(key)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, value).To(Equal(expected), "key %s", key)
}

func statistics(values []float64) (minimum, maximum, average float64) {
	minimum = values[0]
	maximum = values[0]
	var sum float64
	for _, value := range values {
		minimum = min(minimum, value)
		maximum = max(maximum, value)
		sum += value
	}
	return minimum, maximum, sum / float64(len(values))
}
