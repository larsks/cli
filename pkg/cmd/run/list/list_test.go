package list

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

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
	tests := []struct {
		name    string
		opts    *ListOptions
		wantOut string
		stubs   func(*httpmock.Registry)
		nontty  bool
	}{
		/*
			{
				name: "blank tty",
				opts: &ListOptions{
					Limit: defaultLimit,
				},
				stubs: func(reg *httpmock.Registry) {
					reg.Register(
						// TODO parameterize limit
						httpmock.REST("GET", "repos/OWNER/REPO/actions/runs?per_page=10"),
						// TODO fill
						httpmock.JSONResponse(RunsPayload{}),
					)
				},
				wantOut: "TODO",
			},
			{
				name: "blank nontty",
				opts: &ListOptions{
					Limit: defaultLimit,
				},
				nontty:  true,
				wantOut: "TODO",
			},
			{
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
