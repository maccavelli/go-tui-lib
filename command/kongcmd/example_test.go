package kongcmd_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/alecthomas/kong"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/kongcmd"
)

type greetArgs struct {
	Name string `json:"name" arg:"" help:"who to greet"`
}

func greeter() *command.Registry {
	r := command.NewRegistry()
	greet, err := command.New("greet.one", "Greet someone", func(_ context.Context, _ *command.Invocation, a greetArgs) (command.Result, error) {
		return command.Result{Value: map[string]string{"greeting": "hello " + a.Name}, Text: "hello " + a.Name}, nil
	}, command.WithDanger(command.ReadOnly), command.WithCategory("Greetings"))
	if err != nil {
		panic(err)
	}
	if err := r.Register(greet); err != nil {
		panic(err)
	}
	return r
}

// A program builds one registry, and parses its commands with Kong.
func ExampleAdapter_Parser() {
	a, err := kongcmd.New(greeter(), kongcmd.WithName("demo"))
	if err != nil {
		panic(err)
	}
	var out, errOut bytes.Buffer
	p, err := a.Parser(kong.Writers(&out, &errOut))
	if err != nil {
		panic(err)
	}
	for _, args := range [][]string{{"greet", "one", "world"}, {"greet.one", "world", "--json"}, {"list"}} {
		out.Reset()
		_, code, handled := a.Run(context.Background(), p, args)
		fmt.Printf("exit %d handled %v\n%s", code, handled, out.String())
	}
	// Output:
	// exit 0 handled true
	// hello world
	// exit 0 handled true
	// {"greeting":"hello world"}
	// exit 0 handled true
	//
	// Greetings commands:
	//   greet one         Greet someone
	//
	// Other commands:
	//   command describe  Describes one command: its ID, arguments schema and danger.
	//   command list      Lists the commands the caller can run now: not hidden,
	//                     offered where it asks from, and available in its context.
}

// serveCmd is a program's own Kong command, beside the registry's.
type serveCmd struct{}

func (serveCmd) Run(r *command.Registry, ctx context.Context) error {
	_, err := r.Run(ctx, command.Request{ID: "greet.one", Args: []byte(`{"name":"from serve"}`), Origin: command.OriginProgram})
	return err
}

// A program with its own Kong grammar mounts the registry's commands
// beside its own, and runs its own command from the context Run returns.
func ExampleAdapter_Options() {
	a, err := kongcmd.New(greeter(), kongcmd.WithName("demo"))
	if err != nil {
		panic(err)
	}
	var cli struct {
		Serve serveCmd `cmd:"" help:"Serve"`
	}
	var out bytes.Buffer
	p, err := kong.New(&cli, append(a.Options(), kong.Writers(&out, &out))...)
	if err != nil {
		panic(err)
	}
	kctx, code, handled := a.Run(context.Background(), p, []string{"serve"})
	fmt.Println(code, handled, kctx.Command())
	fmt.Println(kctx.Run())
	_, code, handled = a.Run(context.Background(), p, []string{"greet", "one", "world"})
	fmt.Printf("%d %v %s", code, handled, out.String())
	// Output:
	// 0 false serve
	// <nil>
	// 0 true hello world
}
