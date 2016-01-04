package main

import (
  "testing"
)

func TestParseValidDesingDocumentName(t *testing.T) {

  name, err := designDocumentName("_design/v1")
  if err != nil {
    t.Fatalf("Did not extract the design document name")
  }
  if name != "v1" {
    t.Fatalf("Design document name expected to be 'v1' but got: %s", name)
  }
}

func TestParseInvalidDesingDocumentName(t *testing.T) {
  name, err := designDocumentName("design/v1")
  if err != nil {
    t.Fatalf("Did not extract the design document name")
  }
  if name != "design/v1" {
    t.Fatalf("name expected to be 'design/v1' but got: %s", name)
  }
}
