package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type Host struct {
	jobs        []Job
	channel     chan JobDone
	sum         int64
	curRoutines int
	maxRoutines int
}

type JobDone struct {
	newJobs []Job
	size    int64
}

type Job struct {
	dir string
}

func main() {
	numRoutines := flag.Int("t", 1, "The number of threads to use.")
	// summary := flag.Bool("s", false, "Enable if you only want a summary.")
	// format := flag.String("f", "b", "The format of the output.")

	flag.Usage = func() {
		fmt.Println("Welcome to the gdu!")
		fmt.Println("\nUsage:")
		fmt.Printf("  %s [options] [...dirs]\n", os.Args[0])
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
	}

	if *numRoutines < 1 {
		flag.Usage()
		os.Exit(1)
	}

	flag.Parse()

	tail := flag.Args() // The directories to search

	// Actually launch the program
	if len(tail) > 0 {
		// Use the provided directories
		host := CreateHost(tail, *numRoutines)
		if err := host.EvaluatePaths(); err != nil {
			log.Fatal(err)
		}
		fmt.Println(host.sum)

	} else {
		// Use the current directory
		host := CreateHost([]string{"."}, *numRoutines)
		if err := host.EvaluatePaths(); err != nil {
			log.Fatal(err)
		}

		fmt.Println(host.sum)
	}
}

func CreateHost(dirs []string, maxRoutines int) *Host {
	jobs := make([]Job, len(dirs))
	for i, t := range dirs {
		d, err := filepath.Abs(t)
		if err != nil {
			log.Fatal(err)
		}
		jobs[i] = Job{
			dir: d,
		}
	}
	return &Host{
		jobs:        jobs,
		maxRoutines: maxRoutines,
		channel:     make(chan JobDone, maxRoutines),
	}
}

func (h *Host) EvaluatePaths() error {
	for {
		for {
			if h.curRoutines < h.maxRoutines && len(h.jobs) > 0 {
				h.curRoutines++
				newJob := h.jobs[len(h.jobs)-1]
				h.jobs = h.jobs[:len(h.jobs)-1]

				go func(path string) {
					foundJobs, err := EvaluatePath(path)
					if err != nil {
						fmt.Printf("%s error: %v", path, err)
					} else {
						h.channel <- *foundJobs
					}
				}(newJob.dir)
			} else {
				break
			}
		}
		val, ok := <-h.channel
		if ok {
			h.sum += val.size
			h.curRoutines--
			h.jobs = append(h.jobs, val.newJobs...)
		}
		if len(h.jobs) == 0 && h.curRoutines == 0 {
			break
		}
	}
	return nil
}

func EvaluatePath(filePath string) (*JobDone, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()

	entries, err := file.ReadDir(0)
	if err != nil {
		return nil, err
	}

	jobs := []Job{}
	for _, e := range entries {
		// fmt.Println(e)
		if e.IsDir() {
			jobs = append(jobs, Job{filePath + "/" + e.Name()})
		} else {
			info, err := e.Info()
			if err == nil {
				size += info.Size()
			}
		}
	}

	// fmt.Println(size, filePath)
	return &JobDone{
		size:    size,
		newJobs: jobs,
	}, nil
}
