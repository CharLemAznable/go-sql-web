package main

import (
    "bufio"
    "bytes"
    "errors"
    "io"
    "os/exec"
    "strings"
)

// 执行Shell脚本，返回行解析对象数组
func ExecuteBashLiner(shellScripts string, liner func(line string) bool) error {
    cmd := exec.Command("bash", "-c", shellScripts)
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return err
    }

    var eout bytes.Buffer
    cmd.Stderr = &eout

    if err := cmd.Start(); err != nil {
        return err
    }

    defer cmd.Process.Kill() // nolint
    defer cmd.Wait()         // nolint

    reader := bufio.NewReader(stdout)
    for {
        line, err := reader.ReadString('\n')
        if err != nil || err == io.EOF {
            break
        }

        line = strings.TrimSpace(line)
        if line != "" {
            if !liner(line) {
                return nil
            }
        }
    }

    eoutput := eout.String()
    if eoutput != "" {
        return errors.New(eoutput)
    }

    return nil
}

func ExecuteBash(shellScripts string) (string, error) {
    stdout := ""

    err := ExecuteBashLiner(shellScripts, func(line string) bool {
        stdout += line
        return true
    })

    return stdout, err
}
