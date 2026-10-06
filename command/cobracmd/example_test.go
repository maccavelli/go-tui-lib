package cobracmd_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cobracmd"
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

// A program builds one registry, and offers its commands as a Cobra tree.
func ExampleNew() {
	root, err := cobracmd.New(greeter(), cobracmd.WithName("demo"))
	if err != nil {
		panic(err)
	}
	for _, args := range [][]string{{"greet", "one", "world"}, {"greet.one", "world", "--json"}, {"list"}} {
		var out, errOut bytes.Buffer
		code := cobracmd.Run(context.Background(), root, args, nil, &out, &errOut)
		fmt.Printf("exit %d\n%s", code, out.String())
	}
	// Output:
	// exit 0
	// hello world
	// exit 0
	// {"greeting":"hello world"}
	// exit 0
	//
	// Greetings commands:
	//   greet one         Greet someone
	//
	// Other commands:
	//   command describe  Describes one command: its ID, arguments schema and danger.
	//   command list      Lists the commands the caller can run now: not hidden,
	//                     offered where it asks from, and available in its context.
}

// A program with its own Cobra tree mounts the registry's commands beside
// its own.
func ExampleMount() {
	root := &cobra.Command{Use: "demo"}
	root.AddCommand(&cobra.Command{Use: "serve", Short: "Serve", RunE: func(c *cobra.Command, _ []string) error {
		c.Println("serving")
		return nil
	}})
	if err := cobracmd.Mount(root, greeter(), cobracmd.WithName("demo")); err != nil {
		panic(err)
	}
	for _, args := range [][]string{{"serve"}, {"greet", "one", "world"}} {
		var out, errOut bytes.Buffer
		code := cobracmd.Run(context.Background(), root, args, nil, &out, &errOut)
		fmt.Printf("exit %d\n%s%s", code, out.String(), errOut.String())
	}
	// Output:
	// exit 0
	// serving
	// exit 0
	// hello world
}
