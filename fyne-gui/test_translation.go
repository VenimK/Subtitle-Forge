package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// TestTranslation is a helper to test the improved translation
func TestTranslation() {
	fmt.Println("🧪 AI Translation Test Helper")
	fmt.Println("================================")
	fmt.Println()

	// Example: Test with a simple SRT file
	testSRT := `1
00:00:01,000 --> 00:00:03,000
Hello, how are you?

2
00:00:03,500 --> 00:00:05,000
I'm doing great, thanks!

3
00:00:05,500 --> 00:00:08,000
Would you like to grab coffee?`

	// Create a temporary test file
	tempDir := os.TempDir()
	inputFile := filepath.Join(tempDir, "test_input.srt")
	outputFile := filepath.Join(tempDir, "test_output_portuguese.srt")

	err := os.WriteFile(inputFile, []byte(testSRT), 0644)
	if err != nil {
		fmt.Printf("❌ Error creating test file: %v\n", err)
		return
	}

	fmt.Printf("📁 Test input file: %s\n", inputFile)
	fmt.Printf("📁 Test output file: %s\n", outputFile)
	fmt.Println()

	// Get API key from environment or prompt user
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  No GEMINI_API_KEY environment variable found")
		fmt.Println("💡 Set it with: export GEMINI_API_KEY='your-api-key-here'")
		fmt.Println("🔗 Get your API key at: https://aistudio.google.com/app/apikey")
		return
	}

	// Test with GST
	fmt.Println("🚀 Testing with GST...")
	fmt.Println()

	config := AITranslationConfig{
		GSTPath:     findGSTPath(),
		APIKey:      apiKey,
		Model:       "gemini-2.5-flash",
		Temperature: 0.3, // Lower for more consistent translations
		BatchSize:   100,
		Description: "Casual conversation between friends. Use natural, conversational Portuguese.",
		ResumeMode:  false,
		ProgressLog: false,
		ThoughtsLog: false,
	}

	success, errMsg := translateSubtitleFileWithError(inputFile, outputFile, "English", "Portuguese", config)

	if success {
		fmt.Println("✅ Translation completed successfully!")
		fmt.Println()

		// Read and display the output
		outputContent, err := os.ReadFile(outputFile)
		if err == nil {
			fmt.Println("📄 Translation Result:")
			fmt.Println("─────────────────────")
			fmt.Println(string(outputContent))
			fmt.Println("─────────────────────")
		}

		fmt.Println()
		fmt.Println("💡 Try different settings:")
		fmt.Println("   • Lower temperature (0.1-0.3) for more consistent translations")
		fmt.Println("   • Add specific context in Description field")
		fmt.Println("   • Test with different target languages")
		fmt.Println()
		fmt.Printf("📂 Output saved to: %s\n", outputFile)
	} else {
		fmt.Printf("❌ Translation failed: %s\n", errMsg)
	}

	fmt.Println()
	fmt.Println("🔍 What's different in the improved version:")
	fmt.Println("   ✓ Structured JSON request/response with schema")
	fmt.Println("   ✓ Safety settings disabled (BLOCK_NONE) for unrestricted translation")
	fmt.Println("   ✓ Detailed instruction prompts like the Python version")
	fmt.Println("   ✓ Better error handling and validation")
	fmt.Println("   ✓ Support for custom descriptions and context")
}

// To use this test:
// 1. Set your API key: export GEMINI_API_KEY='your-api-key-here'
// 2. Call TestTranslation() from within your main application
// 3. Or add a CLI flag in main.go to trigger this test
