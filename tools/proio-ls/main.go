package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"

	"github.com/proio-org/go-proio"
	_ "github.com/proio-org/go-proio-pb/model/eic"
	_ "github.com/proio-org/go-proio-pb/model/example"
	_ "github.com/proio-org/go-proio-pb/model/lcio"
	_ "github.com/proio-org/go-proio-pb/model/mc"
)

var (
	ignore        = flag.Bool("i", false, "ignore the specified tags instead of isolating them")
	event         = flag.Int64("e", -1, "list specified event, numbered consecutively from the start of the stream starting with 0")
	printMetadata = flag.Bool("m", false, "print metadata as string")
)

func printUsage() {
	fmt.Fprintf(os.Stderr,
		`Usage: proio-ls [options] <proio-input-file> [tags...]

proio-ls will list the contents of a proio stream.  For each event, the tags
are listed in alphabetical order followed by all entries with that tag (this
means that entries with multiple tags will be printed multiple times).
Optionally, tags can be specified, in which case only those tags will be shown.
The -i flag can be specified to ignore the specified tags, instead of isolating
them.  The -e flag can be used to isolate a specific event by its index.

options:
`,
	)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = printUsage
	flag.Parse()

	if flag.NArg() < 1 {
		printUsage()
		log.Fatal("Invalid arguments")
	}

	var reader *proio.Reader
	var err error

	filename := flag.Arg(0)
	if filename == "-" {
		stdin := bufio.NewReader(os.Stdin)
		reader = proio.NewReader(stdin)
	} else {
		reader, err = proio.Open(filename)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	singleEvent := false
	startingEvent := uint64(0)
	if *event >= 0 {
		singleEvent = true
		startingEvent = uint64(*event)
		totalSkipped := uint64(0)
		for {
			var nSkipped uint64
			if nSkipped, err = reader.Skip(startingEvent - totalSkipped); err == io.EOF {
				log.Fatal(err)
			}
			totalSkipped += nSkipped
			if totalSkipped == startingEvent {
				break
			}
		}
	}

	argTags := make(map[string]bool)
	for i := 1; i < flag.NArg(); i++ {
		argTags[flag.Arg(i)] = true
	}

	nEventsRead := uint64(0)
	lastMetadata := make(map[string][]byte)

rescanLoop:
	for {
		var event *proio.Event
		for event = range reader.ScanEvents(1) {
			if *ignore {
				for tag := range argTags {
					event.DeleteTag(tag)
				}
			} else if len(argTags) > 0 {
				for _, tag := range event.Tags() {
					if !argTags[tag] {
						event.DeleteTag(tag)
					}
				}
			}

			if !reflect.DeepEqual(event.Metadata, lastMetadata) {
				fmt.Println("========== META DATA ==========")
				for key, bytes := range event.Metadata {
					fmt.Printf("%v: ", key)
					if *printMetadata {
						fmt.Println(string(bytes))
					} else {
						fmt.Printf("%v bytes\n", len(bytes))
					}
				}
				fmt.Println()
				lastMetadata = event.Metadata
			}

			fmt.Println("========== EVENT", nEventsRead+startingEvent, "==========")
			fmt.Print(event)

			nEventsRead++
			if singleEvent {
				break rescanLoop
			}
		}

		if reader.Err == io.EOF {
			break
		} else {
			log.Print(reader.Err)
		}
	}
}
