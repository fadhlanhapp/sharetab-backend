package main

import (
	"fmt"
	"os/exec"
)

func main() {
	fmt.Println("Testing Go build...")
	
	cmd := exec.Command("go", "build", "-v", ".")
	output, err := cmd.CombinedOutput()
	
	fmt.Printf("Build output:\n%s\n", string(output))
	
	if err != nil {
		fmt.Printf("Build failed: %v\n", err)
	} else {
		fmt.Println("Build successful!")
	}
}