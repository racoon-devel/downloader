package protocol

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Response string

const Ok Response = "ok"

func (r Response) IsOk() bool {
	return r == Ok || r == Ok+" "
}

func (r Response) Write(wr io.Writer) error {
	_, err := wr.Write([]byte(r + "\n"))
	return err
}

func MakeBadResponse(reason string) Response {
	return Response(fmt.Sprintf("fail: \"%s\"", reason))
}

func MakeResponse(args ...string) Response {
	return Ok + " " + Response(fmt.Sprint(args))
}

func ReadResponse(rd io.Reader) (Response, error) {
	brd := bufio.NewReader(rd)
	resp, err := brd.ReadString('\n')
	if err != nil {
		return "", err
	}
	return Response(strings.TrimSuffix(resp, "\n")), nil
}
