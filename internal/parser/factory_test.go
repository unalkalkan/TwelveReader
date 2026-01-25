package parser

import (
	"testing"
)

func TestFactory(t *testing.T) {
	factory := NewFactory()

	t.Run("Get TXT parser", func(t *testing.T) {
		parser, err := factory.GetParser("txt")
		if err != nil {
			t.Fatalf("Failed to get txt parser: %v", err)
		}
		if parser == nil {
			t.Fatal("Got nil parser")
		}
	})

	t.Run("Get PDF parser", func(t *testing.T) {
		parser, err := factory.GetParser("pdf")
		if err != nil {
			t.Fatalf("Failed to get pdf parser: %v", err)
		}
		if parser == nil {
			t.Fatal("Got nil parser")
		}
	})

	t.Run("Get ePUB parser", func(t *testing.T) {
		parser, err := factory.GetParser("epub")
		if err != nil {
			t.Fatalf("Failed to get epub parser: %v", err)
		}
		if parser == nil {
			t.Fatal("Got nil parser")
		}
	})

	t.Run("Case insensitive", func(t *testing.T) {
		parser1, err1 := factory.GetParser("TXT")
		parser2, err2 := factory.GetParser("txt")
		
		if err1 != nil || err2 != nil {
			t.Fatal("Factory should be case insensitive")
		}
		
		if parser1 == nil || parser2 == nil {
			t.Fatal("Got nil parser")
		}
	})

	t.Run("Unsupported format", func(t *testing.T) {
		_, err := factory.GetParser("doc")
		if err == nil {
			t.Error("Expected error for unsupported format")
		}
	})
}
