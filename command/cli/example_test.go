package cli_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/maccavelli/go-tui-lib/command"
	"github.com/maccavelli/go-tui-lib/command/cli"
)

type greetArgs struct {
	Name string `json:"name" arg:"" help:"who to greet"`
}

// A program builds one registry, and offers its commands from the shell.
func ExampleRun() {
	r := command.NewRegistry()
	greet, err := command.New("greet", "Greet someone", func(_ context.Context, _ *command.Invocation, a greetArgs) (command.Result, error) {
		return command.Result{Value: map[string]string{"greeting": "hello " + a.Name}, Text: "hello " + a.Name}, nil
	}, command.WithDanger(command.ReadOnly))
	if err != nil {
		panic(err)
	}
	if err := r.Register(greet); err != nil {
		panic(err)
	}
	var out, errOut bytes.Buffer
	code := cli.Run(context.Background(), r, []string{"list"}, &out, &errOut, cli.WithName("demo"))
	fmt.Printf("exit %d\n%s", code, out.String())
	out.Reset()
	code = cli.Run(context.Background(), r, []string{"greet", "world", "--json"}, &out, &errOut, cli.WithName("demo"))
	fmt.Printf("exit %d\n%s", code, out.String())
	// Output:
	// exit 0
	//
	// Other commands:
	//   command describe  Describes one command: its ID, arguments schema and danger.
	//   command list      Lists the commands the caller can run now: not hidden,
	//                     offered where it asks from, and available in its context.
	//   greet             Greet someone
	// exit 0
	// {"greeting":"hello world"}
}
