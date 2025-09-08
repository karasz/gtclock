package cmd

import (
	"testing"
)

func TestGTAILocal(t *testing.T) {
	mp := make(map[string]string)
	mp["@40000000433225833b6e1a8c"] = "2005-09-22 03:30:33.9970715 +0000 UTC"
	mp["@40000000433225833b6e2644"] = "2005-09-22 03:30:33.9970745 +0000 UTC"
	mp["@40000000433225840c85ba04"] = "2005-09-22 03:30:34.2100905 +0000 UTC"
	mp["@40000000433225840c8f0cbc"] = "2005-09-22 03:30:34.2107015 +0000 UTC"
	mp["@40000000433225852a9ada4c"] = "2005-09-22 03:30:35.7147915 +0000 UTC"
	mp["@"] = "@"
	mp["@452452"] = "@452452"
	mp["@40000000gsdf fgsfdgsfdg"] = "@40000000gsdf fgsfdgsfdg"
	mp["@400000005A848EAD"] = "2018-02-14 19:31:10 +0000 UTC"

	for i, k := range mp {
		if z := processline(i); z != k {
			t.Errorf("Line %s was translated to %s instead of %s", i, z, k)
		}
	}
}
