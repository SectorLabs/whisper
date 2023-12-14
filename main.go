package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"gopkg.in/yaml.v2"
)

var Version string = "dev"
var CommitHash string = "n/a"
var BuildTimestamp string = "n/a"

func main() {
	format := new(string)
	version := new(bool)
	paramsType := new(string)

	flag.StringVar(format, "format", "json", "")
	flag.StringVar(format, "f", "json", "")

	flag.BoolVar(version, "v", false, "")
	flag.BoolVar(version, "version", false, "")

	flag.StringVar(paramsType, "t", "", "")
	flag.StringVar(paramsType, "type", "", "")

	flag.Usage = func() {
		fmt.Printf(`Recursively gather parameters stored in AWS SSM under a given path.

%[1]s [-h/--help] [-f/--format yaml|json] [-t/--type String|SecureString] [path]

Options:
  -h, --help                        show brief help
  -v, --version                     show the current build version
  -f, --format [json|yaml]          output format (default "json")
  -t, --type [String|SecureString]  gather parameter of specific type
`,
			os.Args[0],
		)
	}

	flag.Parse()

	if *version {
		fmt.Printf("Version: %v\nCommit Hash: %v\nBuild Timestamp: %v\n", Version, CommitHash, BuildTimestamp)
		return
	}

	var path = flag.Arg(0)

	if *paramsType != "" && *paramsType != "String" && *paramsType != "SecureString" {
		fmt.Fprintf(
			os.Stderr,
			"Invalid value '%v' for 'type' argument. Accepted values: ['', 'String', 'SecureString'].\n",
			*paramsType,
		)
		os.Exit(1)
	}

	if *format != "json" && *format != "yaml" {
		fmt.Fprintf(
			os.Stderr,
			"Invalid value '%v' for 'format' option. Accepted values: ['yaml', 'json'].\n",
			*format,
		)
		os.Exit(1)
	}

	path = strings.TrimPrefix(strings.TrimSuffix(path, "/"), "/")

	key := fmt.Sprintf("/%s", path)

	client, err := newClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var types []string
	switch *paramsType {
	case "":
		types = []string{"String", "SecureString"}
	case "String":
		types = []string{"String"}
	case "SecureString":
		types = []string{"SecureString"}
	}

	params, err := getParameters(client, key, types)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var result []byte
	switch *format {
	case "json":
		if result, err = json.Marshal(params); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "yaml":
		if result, err = yaml.Marshal(params); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("%v\n", string(result))
}

// newClient creates a new AWS SSM client using the AWS default credentials chain
func newClient() (*ssm.SSM, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			MaxRetries: aws.Int(1),
		},
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		return nil, fmt.Errorf("could not initiate the client: %v", err)
	}

	return ssm.New(sess), nil
}

// getParameters fetches all parameters whose names start with a given key
func getParameters(client *ssm.SSM, key string, types []string) (map[string]any, error) {
	var values []*string
	for _, t := range types {
		values = append(values, aws.String(t))
	}

	var withDecryption bool = false
	for _, t := range types {
		if t == "SecureString" {
			withDecryption = true
			break
		}
	}

	in := ssm.GetParametersByPathInput{
		Path:             aws.String(key),
		Recursive:        aws.Bool(true),
		WithDecryption:   aws.Bool(withDecryption),
		ParameterFilters: []*ssm.ParameterStringFilter{{Key: aws.String("Type"), Values: values}},
	}

	var out ssm.GetParametersByPathOutput

	if err := client.GetParametersByPathPages(&in, func(o *ssm.GetParametersByPathOutput, lastPage bool) bool {
		if o != nil && len(o.Parameters) > 0 {
			out.Parameters = append(out.Parameters, o.Parameters...)
			return true
		}

		return false
	}); err != nil {
		return nil, fmt.Errorf("could not fetch parameters: %v", err)
	}

	return parseParameters(out.Parameters, key), nil
}

// parseParameters converts a list of ssm.Parameter into a map structure, based on the parameters' path
//
// Modified https://github.com/helmfile/vals/blob/b4e9a527ea02d24713761efa5369984beebbc9f2/pkg/providers/ssm/ssm.go#L166
func parseParameters(parameters []*ssm.Parameter, key string) map[string]any {
	res := map[string]any{}

	for _, param := range parameters {
		name := *param.Name

		if key != "/" {
			name = strings.TrimPrefix(name, key)
		}

		if name[0] != '/' {
			panic(fmt.Errorf("bug: unexpected format of parameter: %s in %s must start with a slash(/)", name, *param.Name))
		}

		name = name[1:]

		var current map[string]any = res

		parts := strings.Split(name, "/")
		for i, n := range parts {
			if i == len(parts)-1 {
				current[n] = *param.Value
			} else {
				if m, ok := current[n]; !ok {
					current[n] = map[string]any{}
				} else if _, isMap := m.(map[string]any); !isMap {
					current[n] = map[string]any{}
				}

				current = current[n].(map[string]any)
			}
		}
	}

	return res
}
