package protocol

import (
	"bufio"
	"io"
	"strings"
)

type Command string

const (
	Invalid Command = ""
	Status  Command = "status"
	Task    Command = "task"
	Stop    Command = "stop"
	Done    Command = "done"
)

func (c Command) Split() []string {
	return strings.Split(string(c), " ")
}

func (c Command) Id() string {
	return c.Split()[0]
}

func (c Command) IsValid() bool {
	if c == "" {
		return false
	}

	args := c.Split()
	switch args[0] {
	case "status":
	case "task":
		if len(args) != 2 {
			return false
		}
	case "stop":
	case "done":
	default:
		return false
	}

	return true
}

func (c Command) Args() []string {
	args := c.Split()
	return args[1:]
}

func (c Command) Write(wr io.Writer) error {
	_, err := wr.Write([]byte(c + "\n"))
	return err
}

func ReadCommand(reader io.Reader) (Command, error) {
	rd := bufio.NewReader(reader)
	s, err := rd.ReadString('\n')
	if err != nil {
		return Invalid, err
	}
	return Command(strings.TrimSuffix(s, "\n")), nil
}

func MakeCommand(command Command, args ...string) Command {
	cmd := command
	for _, v := range args {
		cmd += " " + Command(v)
	}
	return cmd
}
