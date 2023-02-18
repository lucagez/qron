package main

// A simple program demonstrating the text input component from the Bubbles
// component library.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucagez/qron/testutil"
)

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type errMsg error

type model struct {
	textarea textarea.Model
	query    string
	err      error
	dbErr    error
	db       *pgxpool.Pool
	teardown func()
	result   string
}

func initialModel() model {
	ti := textarea.New()
	ti.Placeholder = "Query..."
	ti.Focus()
	db, teardown := testutil.PG.CreateDb("scratch")

	return model{
		textarea: ti,
		err:      nil,
		db:       db,
		teardown: teardown,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlA:
			m.dbErr = nil
			m.query = m.textarea.Value()
			rows, err := m.db.Query(context.Background(), m.query)
			if err != nil {
				m.dbErr = err
				break
			}
			var acc string
			for rows.Next() {
				value, err := rows.Values()
				if err != nil {
					m.dbErr = err
					break
				}
				buf, _ := json.Marshal(value)
				acc += "\n"
				acc += string(buf)
			}
			m.result = acc

		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyCtrlE:
			fmt.Println("reseeding 🌱")
			m.teardown()
			db, teardown := testutil.PG.CreateDb("scratch")
			m.db = db
			m.teardown = teardown
			return m, tea.Batch(tea.ClearScreen)

		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	psql := fmt.Sprintf(
		"connect to: psql -d postgres://postgres:postgres@localhost:%d/postgres \n\n",
		m.db.Config().ConnConfig.Port,
	)
	if m.result != "" {
		return fmt.Sprint(
			psql,
			fmt.Sprintf("result: %s\n\n", m.result),
			m.textarea.View()+"\n\n",
		) + "\n\n"
	}
	if m.dbErr != nil {
		return fmt.Sprint(
			psql,
			fmt.Sprintf("error: %v\n\n", m.dbErr),
			m.textarea.View()+"\n\n",
		) + "\n\n"
	}
	return fmt.Sprint(
		psql,
		m.textarea.View()+"\n\n",
	) + "\n\n"
}
