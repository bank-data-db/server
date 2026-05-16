package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/shadiestgoat/bankDataDB/config"
	"github.com/shadiestgoat/bankDataDB/db"
	"github.com/shadiestgoat/bankDataDB/internal"
	"golang.org/x/term"
)

func prompt(p string, isPass bool) string {
	fmt.Print(p + ": ")
	var (
		err error
		inp []byte
	)
	if isPass {
		inp, err = term.ReadPassword(syscall.Stdin)
	} else {
		_, err = fmt.Scan(&inp)
	}

	if err != nil {
		panic("Can't scan: " + err.Error())
	}
	if isPass {
		fmt.Println("")
	}

	return string(inp)
}

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		panic("Usage: ./cmd/user {username}")
	}

	pass1 := prompt("Password", true)
	if len(pass1) < 10 {
		panic("Need at least 10 characters, silly :3")
	}

	pass2 := prompt("Password (confirm)", true)
	if pass1 != pass2 {
		panic("Passwords don't fucking match >:(")
	}

	if prompt(fmt.Sprintf("Great, so shall we proceed to add this user %s (yes/no)?", os.Args[1]), false) != "yes" {
		panic("Cancelling...")
	}

	fmt.Println("Slay (connecting to DB)")

	config.LoadForCLI(true)

	fmt.Println("Slay x2 (alls good)")

	db := db.GetDB(slog.Default())
	id, err := internal.CreateUser(context.Background(), db, os.Args[1], pass1)
	if err != nil {
		panic(err)
	}

	fmt.Println("User made! ID:", id)
}
