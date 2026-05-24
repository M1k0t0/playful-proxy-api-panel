package managementasset

import "testing"

func TestResolveReleaseURLPreservesPinnedGitHubReleaseAPIURL(t *testing.T) {
	input := "https://api.github.com/repos/router-for-me/Cli-Proxy-API-Management-Center/releases/313351637"

	if got := resolveReleaseURL(input); got != input {
		t.Fatalf("resolveReleaseURL() = %q, want %q", got, input)
	}
}

func TestResolveReleaseURLPreservesPinnedGitHubReleaseTagAPIURL(t *testing.T) {
	input := "https://api.github.com/repos/daishuge/playful-proxy-api-panel/releases/tags/v6.10.0-ppap.11"

	if got := resolveReleaseURL(input); got != input {
		t.Fatalf("resolveReleaseURL() = %q, want %q", got, input)
	}
}

func TestResolveReleaseURLNormalizesRepositoryURLsToLatestRelease(t *testing.T) {
	cases := map[string]string{
		"https://github.com/daishuge/playful-proxy-api-panel":                    "https://api.github.com/repos/daishuge/playful-proxy-api-panel/releases/latest",
		"https://api.github.com/repos/daishuge/playful-proxy-api-panel":          "https://api.github.com/repos/daishuge/playful-proxy-api-panel/releases/latest",
		"https://api.github.com/repos/daishuge/playful-proxy-api-panel/releases": "https://api.github.com/repos/daishuge/playful-proxy-api-panel/releases/latest",
	}

	for input, want := range cases {
		if got := resolveReleaseURL(input); got != want {
			t.Fatalf("resolveReleaseURL(%q) = %q, want %q", input, got, want)
		}
	}
}
