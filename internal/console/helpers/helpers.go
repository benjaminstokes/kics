package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/Checkmarx/kics/pkg/model"
	"github.com/hashicorp/hcl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var reportGenerators = map[string]func(path, filename string, body interface{}) error{
	"json": printToJSONFile,
}

// ProgressBar represents a Progress
// Writer is the writer output for progress bar
type ProgressBar struct {
	Writer   io.Writer
	label    string
	space    int
	total    float64
	progress chan float64
}

// NewProgressBar initializes a new ProgressBar
// label is a string print before the progress bar
// total is the progress bar target (a.k.a 100%)
// space is the number of '=' characters on each side of the bar
// progress is a channel updating the current executed elements
func NewProgressBar(label string, space int, total float64, progress chan float64) ProgressBar {
	return ProgressBar{
		Writer:   os.Stdout,
		label:    label,
		space:    space,
		total:    total,
		progress: progress,
	}
}

// Start starts to print a progress bar on console
// wg is a wait group to report when progress is done
func (p *ProgressBar) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	// TODO ioutil will be deprecated on go v1.16, so ioutil.Discard should be changed to io.Discard
	if p.Writer != ioutil.Discard {
		var firstHalfPercentage, secondHalfPercentage string
		const hundredPercent = 100
		formmatingString := "\r" + p.label + "[%s %4.1f%% %s]"
		for {
			currentProgress, ok := <-p.progress
			if !ok || currentProgress >= p.total {
				fmt.Fprintf(p.Writer, formmatingString, strings.Repeat("=", p.space), 100.0, strings.Repeat("=", p.space))
				break
			}

			percentage := currentProgress / p.total * hundredPercent
			convertedPercentage := int(math.Round(float64(p.space+p.space) / hundredPercent * math.Round(percentage)))
			if percentage >= hundredPercent/2 {
				firstHalfPercentage = strings.Repeat("=", p.space)
				secondHalfPercentage = strings.Repeat("=", convertedPercentage-p.space) +
					strings.Repeat(" ", 2*p.space-convertedPercentage)
			} else {
				secondHalfPercentage = strings.Repeat(" ", p.space)
				firstHalfPercentage = strings.Repeat("=", convertedPercentage) +
					strings.Repeat(" ", p.space-convertedPercentage)
			}
			fmt.Fprintf(p.Writer, formmatingString, firstHalfPercentage, percentage, secondHalfPercentage)
		}
	}
}

// WordWrap Wraps text at the specified number of words
func WordWrap(s, identation string, limit int) string {
	if strings.TrimSpace(s) == "" {
		return s
	}

	wordSlice := strings.Fields(s)
	var result string

	for len(wordSlice) >= 1 {
		result = result + identation + strings.Join(wordSlice[:limit], " ") + "\r\n"

		wordSlice = wordSlice[limit:]
		if len(wordSlice) < limit {
			limit = len(wordSlice)
		}
	}
	return result
}

// PrintResult prints on output the summary results
func PrintResult(summary *model.Summary, failedQueries map[string]error) error {
	fmt.Printf("Files scanned: %d\n", summary.ScannedFiles)
	fmt.Printf("Parsed files: %d\n", summary.ParsedFiles)
	fmt.Printf("Queries loaded: %d\n", summary.TotalQueries)

	fmt.Printf("Queries failed to execute: %d\n", summary.FailedToExecuteQueries)
	for queryName, err := range failedQueries {
		fmt.Printf("\t- %s:\n", queryName)
		fmt.Printf("%s", WordWrap(err.Error(), "\t\t", 5))
	}
	fmt.Printf("------------------------------------\n")
	for _, q := range summary.Queries {
		fmt.Printf("%s, Severity: %s, Results: %d\n", q.QueryName, q.Severity, len(q.Files))

		for i := 0; i < len(q.Files); i++ {
			fmt.Printf("\t%s:%d\n", q.Files[i].FileName, q.Files[i].Line)
		}
	}
	fmt.Printf("\nResults Summary:\n")
	fmt.Printf("HIGH: %d\n", summary.SeveritySummary.SeverityCounters["HIGH"])
	fmt.Printf("MEDIUM: %d\n", summary.SeveritySummary.SeverityCounters["MEDIUM"])
	fmt.Printf("LOW: %d\n", summary.SeveritySummary.SeverityCounters["LOW"])
	fmt.Printf("INFO: %d\n", summary.SeveritySummary.SeverityCounters["INFO"])
	fmt.Printf("TOTAL: %d\n\n", summary.SeveritySummary.TotalCounter)
	log.
		Info().
		Msgf("\n\nFiles scanned: %d\n"+
			"Parsed files: %d\nQueries loaded: %d\n"+
			"Queries failed to execute: %d\n",
			summary.ScannedFiles, summary.ParsedFiles, summary.TotalQueries, summary.FailedToExecuteQueries)
	log.
		Info().
		Msg("Inspector stopped\n")

	return nil
}

// printToJSONFile prints on JSON file the summary results
func printToJSONFile(path, filename string, body interface{}) error {
	fullPath := filepath.Join(path, filename+".json")
	_ = os.MkdirAll(path, os.ModePerm)
	f, err := os.OpenFile(filepath.Clean(fullPath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Err(err).Msgf("failed to close file %s", fullPath)
		}

		if err != nil {
			log.Err(err).Msgf("failed to get current directory")
		}

		log.Info().Str("fileName", filename+".json").Msgf("Results saved to file %s", fullPath)
	}()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "\t")

	return encoder.Encode(body)
}

// CustomConsoleWriter creates an output to print log in a files
func CustomConsoleWriter(fileLogger *zerolog.ConsoleWriter) zerolog.ConsoleWriter {
	fileLogger.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}

	fileLogger.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}

	fileLogger.FormatErrFieldName = func(i interface{}) string {
		return "ERROR:"
	}

	fileLogger.FormatFieldValue = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}

	return *fileLogger
}

// FileAnalyzer determines the type of extension in the passed config file by its content
func FileAnalyzer(path string) (string, error) {
	ostat, err := os.Open(path)
	if err != nil {
		return "", err
	}
	rc, err := ioutil.ReadAll(ostat)
	if err != nil {
		return "", err
	}
	var temp map[string]interface{}

	if err := json.Unmarshal(rc, &temp); err == nil {
		return "json", nil
	}

	if err := yaml.Unmarshal(rc, &temp); err == nil {
		return "yaml", nil
	}

	if _, err := toml.Decode(string(rc), &temp); err == nil {
		return "toml", nil
	}

	if c, err := hcl.Parse(string(rc)); err == nil {
		if err = hcl.DecodeObject(&temp, c); err == nil {
			return "hcl", nil
		}
	}

	return "", errors.New("invalid configuration file format")
}

// GenerateOutput execute each report function to generate report
func GenerateOutput(path, filename string, body interface{}, formats []string) error {
	var err error = nil
	for _, format := range formats {
		if err = reportGenerators[format](path, filename, body); err != nil {
			break
		}
	}
	return err
}

// ValidateReportFormats returns an error if output format is not supported
func ValidateReportFormats(formats []string) error {
	validFormats := make([]string, len(reportGenerators))
	for reportFormats := range reportGenerators {
		validFormats = append(validFormats, reportFormats)
	}
	for _, format := range formats {
		if _, ok := reportGenerators[format]; !ok {
			return fmt.Errorf(
				fmt.Sprintf("Report format not supported: %s\nSupportted formats:\n  %s\n", format, strings.Join(validFormats, "\n  ")),
			)
		}
	}
	return nil
}
