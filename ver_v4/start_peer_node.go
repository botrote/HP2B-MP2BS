package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	commandA := "./pr_make_tree" // 매개변수는 참여노드의 IP와 port (2개)
	commandB := "./pr_node"

	cmdA := exec.Command(commandA)
	cmdB := exec.Command(commandB)

	cmdA.Stdout = os.Stdout
	cmdA.Stderr = os.Stderr

	cmdB.Stdout = os.Stdout
	cmdB.Stderr = os.Stderr

	if err := cmdA.Start(); err != nil {
		log.Fatal(err)
	}

	if err := cmdB.Start(); err != nil {
		log.Fatal(err)
	}

	if err := cmdA.Wait(); err != nil {
		log.Fatal(err)
	}

	if err := cmdB.Wait(); err != nil {
		log.Fatal(err)
	}
}