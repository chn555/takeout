package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"gopkg.in/yaml.v3"
)

func main() {
	currentOrder := new(Order)
	currentOrder.Extras = make(map[string]string)

	imprt, err := askToImport()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if imprt {
		currentOrder, err = askToimportOrderFromFile()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if currentOrder.MainDish == "" {
		dish, err := selectMainDish()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		currentOrder.MainDish = dish
	}

	if len(currentOrder.Toppings) == 0 {
		top, err := pickToppings(currentOrder.MainDish)
		if err != nil {
			fmt.Println(err)
		}
		currentOrder.Toppings = top
	}

	if currentOrder.MainDish == Hamburger && currentOrder.Extras[cookingLevelKey] == "" {
		level, err := pickCookingLevel()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		currentOrder.Extras[cookingLevelKey] = level
	}

	err = saveOrder(currentOrder)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Your order is done.")

}

type Order struct {
	MainDish MainDish          `json:"main_dish,omitempty" yaml:"main_dish,omitempty"`
	Toppings []string          `json:"toppings,omitempty" yaml:"toppings,omitempty"`
	Extras   map[string]string `json:"extras,omitempty" yaml:"extras,omitempty"`
}

type MainDish string

const cookingLevelKey = "cookingLevel"
const (
	Hamburger MainDish = "Hamburger"
	Pizza     MainDish = "Pizza"
)

var possibleToppings = map[MainDish][]string{
	Hamburger: {"Cheder", "Onion"},
	Pizza:     {"Tuna", "Olives"},
}

func selectMainDish() (MainDish, error) {
	prompt := promptui.Select{
		Label: "Please select a main dish",
		Items: []MainDish{Hamburger, Pizza},
	}

	_, answer, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get response: %w", err)
	}

	return MainDish(answer), nil

}

func askToImport() (bool, error) {
	imprt := "Continue an existing order"
	noImport := "Create a new order"

	importPrompt := promptui.Select{
		Label: "Please select whether to continue an existing order or create a new one",
		Items: []string{imprt, noImport},
	}

	_, answer, err := importPrompt.Run()
	if err != nil {
		return false, fmt.Errorf("failed to get response: %w", err)
	}

	switch answer {
	case imprt:
		return true, nil
	case noImport:
		return false, nil
	default:
		return false, errors.New("failed to understand choice")
	}

}

func pickToppings(dish MainDish) ([]string, error) {
	toppings, ok := possibleToppings[dish]
	if !ok {
		return nil, nil
	}

	selectedToppings := make([]string, 0)

	for {
		prompt := promptui.Select{
			Label: "Please enter a topping from the following list, or select done to continue",
			Items: append(toppings, "Done"),
		}

		i, topping, err := prompt.Run()
		if err != nil {
			return nil, fmt.Errorf("failed to get response: %w", err)
		}

		if topping == "Done" {
			break
		}

		toppings = append(toppings[:i], toppings[i+1:]...)
		selectedToppings = append(selectedToppings, topping)
	}
	return selectedToppings, nil
}

func askToimportOrderFromFile() (*Order, error) {
	validate := func(i string) error {
		path, err := filepath.Abs(i)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for file %w", err)
		}
		f, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}

		if f.IsDir() {
			return errors.New("provided path is a dir, not a file")
		}

		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Please enter the filename of an existing order",
		Validate: validate,
	}

	path, err := prompt.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return readOrderFromFile(f)
}

func readOrderFromFile(file *os.File) (*Order, error) {
	order := new(Order)
	var err error

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	switch filepath.Ext(file.Name()) {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(bytes, order)
	case ".json":
		err = json.Unmarshal(bytes, order)
	default:
		return nil, errors.New("failed to determine file type")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read file contents: %w", err)
	}

	return order, nil
}

func pickCookingLevel() (string, error) {
	prompt := promptui.Select{
		Label: "Please select a cooking level for the hamburger",
		Items: []string{"MR", "M", "MW", "WD"},
	}

	_, level, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("failed to determin cooking level: %w", err)
	}

	return level, nil
}

func saveOrder(order *Order) error {
	f, err := os.CreateTemp("", "order*.yml")
	if err != nil {
		return fmt.Errorf("failed to create order file: %w", err)
	}

	orderYML, err := yaml.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshel to yml: %w", err)
	}

	_, err = f.Write(orderYML)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	fmt.Println("Successfully wrote down order at " + f.Name())
	return nil
}
