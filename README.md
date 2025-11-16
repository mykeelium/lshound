# lshound

This is a package that is used to map out the ACLs of the filesystem on a Unix based OS. You can use this software to generate the OpenGraph representation of a filesystem to see how the files may interact with each other, and what type of control users and groups have. Some other properties of the file system are also collected.

#### Requirements

go version 1.25
OS: Unix based

To install this program, go version 1.25 is required to build. Once this and GOPATH is properly set up, go install could be run. 

#### Install

`go install github.com/mykeelium/lshound@latest`

#### Usage



##### Command Line Arguments

```text
Usage: lshound [Arguments]

Arguments:
    -path <path>        Path to where to recursively walk down files                | Default: .
    -acl                Collect the ACL information about the file                  | Default: false
    -follow-symlink     While doing the walk, whether or not to follow symlinks     | Default: false
    -max-depth <depth>  Max recursive depth relative to start (-1 = unlimited)      | Default: -1
    -stdout             Whether or not to output to standard out                    | Default: false
    -output <fileName>  Specify the file name to output to if not output to stdout  | Default: output
```
