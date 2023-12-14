# whisper

Whisper is a small, lightweight and _blazingly fast_ CLI tool that can recursively gather parameters stored in the AWS Systems Manager Parameter Store under a specific path.

## Installation

Whisper is a single binary that can be deployed to almost any platform. To install it, download the latest binary from the [Releases](https://github.com/SectorLabs/whisper/releases) page.

If you need an installation script:
```bash
curl -sL "https://github.com/SectorLabs/whisper/releases/latest/download/whisper_$(go env GOOS)_$(go env GOARCH)" -o whisper
chmod +x whisper
mkdir -p ~/.local/bin
mv whisper ~/.local/bin/whisper
```

Note: Make sure `$HOME/.local/bin` is in your `$PATH` environment variable.

## Usage

```bash
$ whisper --help
Recursively gather parameters stored in AWS SSM under a given path.

whisper [-h/--help] [-f/--format yaml|json] [-t/--type String|SecureString] [path]

Options:
  -h, --help                        show brief help
  -v, --version                     show the current build version
  -f, --format [json|yaml]          output format (default "json")
  -t, --type [String|SecureString]  gather parameter of specific type
```
### Example

Assuming the following SSM structure (we will ignore the type of the parameters for now):

```
NAME            VALUE
/foo/bar        A
/foo/baz        B
/foo/foo/bar    C
/foo/bar/foo    D
/foo/bar/baz    E
```

If you wish to export all of them, you can run:
```bash
whisper
```
Then, you will receive the following output (which contains all parameters available in your account):
```json
{
  "foo": {
    "baz": "B",
    "foo": {
      "bar": "C"
    },
    "bar": {
      "foo": "D",
      "baz": "E",
    }
  }
}
```

Notice that there was a conflict between `/foo/bar` and `/foo/bar/*` and because of that the attribute `/foo/bar` was dropped in favor of the `/foo/bar/*` attributes.

If you wish to export a subset of those parameters (let's say the ones that are located under `/foo/bar`), you can run:
```bash
whisper foo/bar
```
Then you will receive the following output:
```json
{
  "foo": "D",
  "baz": "E"
}
```

Note: Notice that the key is stripped from the result structure. The object returned contains all the parameters available **under** that specific key.

## Credentials

For simplicity, `whisper` is built to use the [AWS default credentials chain](https://docs.aws.amazon.com/sdk-for-java/latest/developer-guide/credentials-chain.html#credentials-default), so no additional configuration for credentials is available.

You can configure the credentials chain externally, for example:
```bash
export AWS_ACCESS_KEY_ID="access-key-id"
export AWS_SECRET_ACCESS_KEY="secret-access-key"
whisper
```

## Alternatives

If you need a tool with more complex behaviors, consider similar solutions like:

- [vals](https://github.com/helmfile/vals)
- [chamber](https://github.com/segmentio/chamber)
