package cli

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/antchfx/htmlquery"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"

	"github.com/oinume/icecrawl/log"
)

var (
	concurrency int
	interval    time.Duration
	format      string
)

func init() {
	flagSet := scrapeCommand.Flags()
	flagSet.IntVar(&concurrency, "concurrency", 1, "number of concurrent fetches")
	flagSet.DurationVar(&interval, "interval", 500*time.Millisecond, "interval between fetches")
	flagSet.StringVar(&format, "format", "markdown", "output format (markdown or pdf)")
	// TODO: Define `--out-dir` flag
}

var scrapeCommand = &cobra.Command{
	Use:     "scrape",
	Short:   "scrape subcommand scrapes HTML with given URLs and save it as markdown or PDF.",
	Long:    "scrape subcommand scrapes HTML with given URLs and save it as markdown or PDF.",
	Args:    cobra.RangeArgs(1, 10),
	Example: `icecrawl scrape --format markdown https://example.com`,
	RunE:    scrape,
}

var (
	defaultHTTPClient = &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: redirectErrorFunc,
		Transport: &http.Transport{
			MaxIdleConns:        20,
			MaxIdleConnsPerHost: 20,
			Proxy:               http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 300 * time.Second,
			}).DialContext,
			IdleConnTimeout:     300 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				ClientSessionCache: tls.NewLRUClientSessionCache(100),
				MinVersion:         tls.VersionTLS12,
			},
			ExpectContinueTimeout: 2 * time.Second,
		},
	}
	redirectErrorFunc = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
)

func scrape(cmd *cobra.Command, args []string) error {
	//fmt.Printf("scrape command executed: cmd=%+v, parent=%+v, args=%+v\n", cmd.Name(), cmd.Parent().Name(), args)

	ctx := cmd.Context()
	log.FC(ctx).Info("scrape command started", "args", args) // TODO: Use debug level
	startedAt := time.Now().UTC()
	defer func() {
		elapsed := time.Since(startedAt)
		log.FC(ctx).Info(
			"scrape command finished",
			slog.Int("elapsedInMillis", int(elapsed.Milliseconds())),
		)
	}()

	// TODO: Define as func and log the error
	for _, arg := range args {
		u, err := url.Parse(arg)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}

		document, err := fetchDocument(u)
		if err != nil {
			return fmt.Errorf("failed to fetch document: %w", err)
		}

		var result []byte
		switch format {
		case "markdown": // TODO: Define format as enum
			result, err = scrapeAsMarkdown(ctx, document)
			if err != nil {
				return fmt.Errorf("failed to scrape as markdown: %w", err)
			}
			fileName := fmt.Sprintf("%s.md", document.Title) // TODO: Determine file suffix from format
			if err := os.WriteFile(fileName, result, 0644); err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
			log.FC(ctx).Info("scrape command saved file", "url", u.String(), "fileName", fileName)
		case "pdf":
			result, err = scrapeAsPDF(ctx, document)
			if err != nil {
				return fmt.Errorf("failed to scrape as pdf: %w", err)
			}
			fileName := fmt.Sprintf("%s.pdf", document.Title) // TODO: Determine file suffix from format
			if err := os.WriteFile(fileName, result, 0644); err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
			log.FC(ctx).Info("scrape command saved file", "url", u.String(), "fileName", fileName)
		default:
			return fmt.Errorf("invalid format: %s", format)
		}

	}

	return nil
}

type Document struct {
	URL      *url.URL
	Title    string
	RawBody  []byte
	RootNode *html.Node
}

func fetchDocument(u *url.URL) (*Document, error) {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do http request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch page: statusCode=%s", resp.Status)
	}
	defer resp.Body.Close()

	document := &Document{
		URL: u,
	}
	var out bytes.Buffer
	teeReader := io.TeeReader(resp.Body, &out)
	root, err := htmlquery.Parse(teeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	document.RawBody = out.Bytes()
	document.RootNode = root

	titleNode, err := htmlquery.Query(root, `/html/head/title`)
	if err != nil {
		return nil, fmt.Errorf("failed to query title: %w", err)
	}
	document.Title = htmlquery.InnerText(titleNode)

	return document, nil
}

func scrapeAsMarkdown(_ context.Context, document *Document) ([]byte, error) {
	markdown, err := htmltomarkdown.ConvertNode(document.RootNode)
	if err != nil {
		return nil, err
	}
	return markdown, nil
}

func scrapeAsPDF(ctx context.Context, document *Document) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var buffer []byte
	toPDF := chromedp.Tasks{
		chromedp.Navigate(document.URL.String()),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			buffer, _, err = page.PrintToPDF().WithPrintBackground(false).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	}
	err := chromedp.Run(ctx, chromedp.Navigate(document.URL.String()), toPDF)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape as PDF: %w", err)
	}
	return buffer, nil
}
