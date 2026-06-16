package plugins

import "testing"

func TestTrimProjectPrefixNormalizesWindowsPaths(t *testing.T) {
	got := trimProjectPrefix(`F:\go\auto-desk\frontend\node_modules\pkg\index.js`, `F:\go\auto-desk`)
	want := "frontend/node_modules/pkg/index.js"
	if got != want {
		t.Fatalf("trimProjectPrefix() = %q, want %q", got, want)
	}
}

func TestShouldSkipContentSearchPathDefaults(t *testing.T) {
	excludedSegments := contentSearchExcludeSegments()
	cases := []struct {
		name string
		path string
		want bool
	}{
		{name: "deep node modules slash", path: "frontend/node_modules/pkg/index.js", want: true},
		{name: "deep node modules backslash", path: `src\frontend\node_modules\pkg\index.js`, want: true},
		{name: "frontend source is not hardcoded", path: "frontend/collect-ui/src/App.tsx", want: false},
		{name: "source map", path: "server/static/app.js.map", want: true},
		{name: "minified js", path: "server/static/app.min.js", want: true},
		{name: "binary image", path: "public/logo.png", want: true},
		{name: "go source", path: "plugins/module_workspace_content_search.go", want: false},
		{name: "frontend-like segment", path: "src/frontendish/main.ts", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldSkipContentSearchPath(tc.path, excludedSegments)
			if got != tc.want {
				t.Fatalf("shouldSkipContentSearchPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestShouldSkipContentSearchPathUsesCustomExcludedSegmentsAtAnyDepth(t *testing.T) {
	excludedSegments := contentSearchExcludeSegments("frontend,generated-client")
	cases := []string{
		"frontend/collect-ui/assets/app.js",
		"apps/admin/frontend/src/App.tsx",
		`services\api\generated-client\index.ts`,
	}

	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			if !shouldSkipContentSearchPath(path, excludedSegments) {
				t.Fatalf("expected %q to match custom excluded segment at any depth", path)
			}
		})
	}
}

func TestMatchesIncludePatternsNormalizesPaths(t *testing.T) {
	got := matchesIncludePatterns(
		`F:\go\auto-desk\frontend\src\App.tsx`,
		`frontend\src\App.tsx`,
		"App.tsx",
		[]string{"frontend/src"},
	)
	if !got {
		t.Fatal("expected normalized include pattern to match Windows-style path")
	}
}
