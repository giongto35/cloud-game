package manager

import "testing"

func TestCoreUrl(t *testing.T) {
	testAddress := "https://test.me"
	tests := []struct {
		arch     ArchInfo
		compress string
		f        string
		repo     string
		result   string
	}{
		{
			arch:   ArchInfo{Arch: "x86_64", Ext: ".so", Os: "linux"},
			f:      "uber_core",
			repo:   "buildbot",
			result: testAddress + "/" + "linux/x86_64/latest/uber_core.so",
		},
		{
			arch:     ArchInfo{Arch: "x86_64", Ext: ".so", Os: "linux"},
			compress: "zip",
			f:        "uber_core",
			repo:     "buildbot",
			result:   testAddress + "/" + "linux/x86_64/latest/uber_core.so.zip",
		},
		{
			arch:   ArchInfo{Arch: "x86_64", Ext: ".dylib", Os: "osx", Vendor: "apple"},
			f:      "uber_core",
			repo:   "buildbot",
			result: testAddress + "/" + "apple/osx/x86_64/latest/uber_core.dylib",
		},
		{
			arch:   ArchInfo{Os: "linux", Arch: "x86_64", Ext: ".so"},
			f:      "uber_core",
			repo:   "github",
			result: testAddress + "/" + "linux/x86_64/latest/uber_core.so?raw=true",
		},
		{
			arch:     ArchInfo{Os: "linux", Arch: "x86_64", Ext: ".so"},
			compress: "zip",
			f:        "uber_core",
			repo:     "github",
			result:   testAddress + "/" + "linux/x86_64/latest/uber_core.so.zip?raw=true",
		},
		{
			arch:   ArchInfo{Os: "osx", Arch: "x86_64", Vendor: "apple", Ext: ".dylib"},
			f:      "uber_core",
			repo:   "github",
			result: testAddress + "/" + "apple/osx/x86_64/latest/uber_core.dylib?raw=true",
		},
	}

	for _, test := range tests {
		r := NewRepo(test.repo, testAddress, test.compress, "")
		url := r.CoreUrl(test.f, test.arch)
		if url != test.result {
			t.Errorf("seems that expected link address is incorrect (%v) for file %s %+v", url, test.f, test.arch)
		}
	}
}
