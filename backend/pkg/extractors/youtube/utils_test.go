package youtube

import "testing"

func TestParseCaptions(t *testing.T) {

	body := []byte(`
<?xml version="1.0" encoding="utf-8" ?>
  <transcript>
	<text start="16.99" dur="7.029">Hello</text>
	<text start="19.64" dur="7.229">Welcome to use CanMe</text>
	<text start="24.019" dur="4.5">CanMe is an AI-powered subtitle conversion tool</text>
	<text start="26.869" dur="4.681">Hope you have a nice day</text>
	<text start="28.519" dur="5.791">I am appreciate</text>
	<text start="1376.029" dur="3.081">thank you</text>
  </transcript>`)

	captions, err := ParseCaptions(body)
	if err != nil {
		t.Error(err)
	}

	if captions.Items[4].String() != "I am appreciate" {
		t.Error("expected I am appreciate, got", captions.Items[4].String())
	}

}
