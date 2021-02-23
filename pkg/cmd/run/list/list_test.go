package list

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/cli/cli/internal/ghrepo"
	"github.com/cli/cli/pkg/cmd/run/shared"
	"github.com/cli/cli/pkg/cmdutil"
	"github.com/cli/cli/pkg/httpmock"
	"github.com/cli/cli/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		tty      bool
		wants    ListOptions
		wantsErr bool
	}{
		{
			name: "blank",
			wants: ListOptions{
				Limit: defaultLimit,
			},
		},
		{
			name: "limit",
			cli:  "--limit 100",
			wants: ListOptions{
				Limit: 100,
			},
		},
		{
			name:     "bad limit",
			cli:      "--limit hi",
			wantsErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, _, _ := iostreams.Test()
			io.SetStdinTTY(tt.tty)
			io.SetStdoutTTY(tt.tty)

			f := &cmdutil.Factory{
				IOStreams: io,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *ListOptions
			cmd := NewCmdList(f, func(opts *ListOptions) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(ioutil.Discard)
			cmd.SetErr(ioutil.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				return
			}

			assert.Equal(t, tt.wants.Limit, gotOpts.Limit)
		})
	}
}

func TestListRun(t *testing.T) {

	testRun := func(name string, id int, s shared.Status, c shared.Conclusion) shared.Run {
		created, _ := time.Parse("2006-01-02 15:04:05", "2021-02-23 04:51:00")
		updated, _ := time.Parse("2006-01-02 15:04:05", "2021-02-23 04:55:34")
		return shared.Run{
			Name:       name,
			ID:         id,
			CreatedAt:  created,
			UpdatedAt:  updated,
			Status:     s,
			Conclusion: c,
			Event:      "push",
			HeadBranch: "trunk",
			JobsURL:    fmt.Sprintf("runs/%d/jobs", id),
			HeadCommit: shared.Commit{"cool commit"},
			HeadSha:    "1234567890",
			URL:        fmt.Sprintf("runs/%d", id),
		}
	}

	runs := []shared.Run{
		testRun("successful", 1, shared.Completed, shared.Success),
		testRun("in progress", 2, shared.InProgress, ""),
		testRun("timed out", 3, shared.Completed, shared.TimedOut),
		testRun("cancelled", 4, shared.Completed, shared.Cancelled),
		testRun("failed", 5, shared.Completed, shared.Failure),
		testRun("neutral", 6, shared.Completed, shared.Neutral),
		testRun("skipped", 7, shared.Completed, shared.Skipped),
		testRun("requested", 8, shared.Requested, ""),
		testRun("queued", 9, shared.Queued, ""),
		testRun("stale", 10, shared.Completed, shared.Stale),
	}

	tests := []struct {
		name    string
		opts    *ListOptions
		wantOut string
		stubs   func(*httpmock.Registry)
		nontty  bool
	}{
		{
			name: "blank tty",
			opts: &ListOptions{
				Limit: defaultLimit,
			},
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/runs"),
					httpmock.JSONResponse(shared.RunsPayload{
						TotalCount:   10,
						WorkflowRuns: runs,
					}))
			},
			wantOut: "✓  cool commit  successful   trunk  push  1\n-  cool commit  in progress  trunk  push  2\nX  cool commit  timed out    trunk  push  3\n✓  cool commit  cancelled    trunk  push  4\nX  cool commit  failed       trunk  push  5\n✓  cool commit  neutral      trunk  push  6\n✓  cool commit  skipped      trunk  push  7\n-  cool commit  requested    trunk  push  8\n-  cool commit  queued       trunk  push  9\nX  cool commit  stale        trunk  push  10\n\nFor details on a run, try: gh run view <run-id>\n",
		},
		{
			name: "blank nontty",
			opts: &ListOptions{
				Limit:       defaultLimit,
				PlainOutput: true,
			},
			nontty: true,
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/actions/runs"),
					httpmock.JSONResponse(shared.RunsPayload{
						TotalCount:   10,
						WorkflowRuns: runs,
					}))
			},
			wantOut: "completed\tsuccess\tcool commit\tsuccessful\ttrunk\tpush\t4m34s\t1\nin_progress\t\tcool commit\tin progress\ttrunk\tpush\t4m34s\t2\ncompleted\ttimed_out\tcool commit\ttimed out\ttrunk\tpush\t4m34s\t3\ncompleted\tcancelled\tcool commit\tcancelled\ttrunk\tpush\t4m34s\t4\ncompleted\tfailure\tcool commit\tfailed\ttrunk\tpush\t4m34s\t5\ncompleted\tneutral\tcool commit\tneutral\ttrunk\tpush\t4m34s\t6\ncompleted\tskipped\tcool commit\tskipped\ttrunk\tpush\t4m34s\t7\nrequested\t\tcool commit\trequested\ttrunk\tpush\t4m34s\t8\nqueued\t\tcool commit\tqueued\ttrunk\tpush\t4m34s\t9\ncompleted\tstale\tcool commit\tstale\ttrunk\tpush\t4m34s\t10\n",
		},
		/*
			// TODO pagination
				{
					name: "blank nontty",
					opts: &ListOptions{
						Limit: defaultLimit,
					},
					nontty:  true,
					wantOut: "TODO",
				},
				{
					// TODO unclear how to check that limit is properly passed
					name: "respects limit",
					opts: &ListOptions{
						Limit: 1,
					},
					wantOut: "TODO",
				},
		*/
		{
			name: "no results nontty",
			opts: &ListOptions{
				Limit:       defaultLimit,
				PlainOutput: true,
			},
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					//httpmock.REST("GET", "repos/OWNER/REPO/actions/runs?per_page=10?page=1"),
					httpmock.REST("GET", "repos/OWNER/REPO/actions/runs"),
					httpmock.JSONResponse(shared.RunsPayload{}),
				)
			},
			nontty:  true,
			wantOut: "",
		},
		{
			name: "no results tty",
			opts: &ListOptions{
				Limit: defaultLimit,
			},
			stubs: func(reg *httpmock.Registry) {
				reg.Register(
					//httpmock.REST("GET", "repos/OWNER/REPO/actions/runs?per_page=10?page=1"),
					httpmock.REST("GET", "repos/OWNER/REPO/actions/runs"),
					httpmock.JSONResponse(shared.RunsPayload{}),
				)
			},
			wantOut: "No runs found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &httpmock.Registry{}
			tt.stubs(reg)

			tt.opts.HttpClient = func() (*http.Client, error) {
				return &http.Client{Transport: reg}, nil
			}

			io, _, stdout, _ := iostreams.Test()
			io.SetStdoutTTY(!tt.nontty)
			tt.opts.IO = io
			tt.opts.BaseRepo = func() (ghrepo.Interface, error) {
				return ghrepo.FromFullName("OWNER/REPO")
			}

			err := listRun(tt.opts)
			assert.NoError(t, err)

			assert.Equal(t, tt.wantOut, stdout.String())
			reg.Verify(t)
		})
	}
}
