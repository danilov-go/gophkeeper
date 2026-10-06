package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/client/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/client/service"
	"github.com/danilov-go/gophkeeper/internal/client/tui"
)

func main() {
	storage := memory.InitMemStorage()
	cryptoKey := []byte("ase_secret_key_testtesttesttestt")
	client := service.NewClientService(storage, cryptoKey)
	appModel := tui.NewAppModel(client)
	p := tea.NewProgram(appModel)
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
