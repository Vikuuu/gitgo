package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"

	"github.com/Vikuuu/gitgo"
)

// --------------- Helper functions -------------------

type testdata struct {
	dir    string
	stdin  *os.File
	stdout *os.File
	stderr *os.File
}

func cmdInit() *commands {
	cmds := &commands{
		registeredCmds: make(map[string]commandInfo),
	}
	cmds.initializeCommands()

	return cmds
}

func tempFile(name string) *os.File {
	f, err := os.CreateTemp("/tmp", name)
	if err != nil {
		panic(err)
	}
	return f
}

func testDataInit() testdata {
	// make directory in `/tmp`
	tmpDir, err := os.MkdirTemp("/tmp", "gitgo-test")
	if err != nil {
		panic(err)
	}

	// Create standard input, output, and error
	// temporary files
	stdin := tempFile("stdin")
	stdout := tempFile("stdout")
	stderr := tempFile("stderr")

	return testdata{
		dir:    tmpDir,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

func testGitgoVar() map[string]string {
	env := make(map[string]string)
	env["name"] = "Test User"
	env["email"] = "test@example.com"

	return env
}

func testRepoInitialize(t *testing.T, td testdata) command {
	cmds := cmdInit()

	cmd := command{
		name:   "init",
		args:   []string{},
		env:    testGitgoVar(),
		pwd:    td.dir,
		stdin:  td.stdin,
		stdout: td.stdout,
		stderr: td.stderr,
		repo:   gitgo.NewRepository(td.dir),
	}

	exitCode, err := cmds.run(cmd)
	if err != nil {
		t.Log("Cannot initialize test repo.\n")
		panic(err)
	}
	if exitCode != 0 {
		t.Log("Cannot initialize test repo.\n")
		panic(exitCode)
	}

	return cmd
}

func tearUp(t *testing.T) (*commands, command) {
	td := testDataInit()
	cmds := cmdInit()

	// initialize `.gitgo` in test repository
	cmd := testRepoInitialize(t, td)

	return cmds, cmd
}

func tearUpWithoutRepo(_ *testing.T) (*commands, testdata) {
	td := testDataInit()
	cmds := cmdInit()

	return cmds, td
}

func tearDown(t *testing.T, cmd command) {
	err := os.RemoveAll(cmd.pwd)
	assert.NoErrorf(t, err, "teardown failed: err removing test dir")
	err = os.Remove(cmd.stdin.Name())
	assert.NoErrorf(t, err, "teardown failed: err removing test stdin")
	err = os.Remove(cmd.stderr.Name())
	assert.NoErrorf(t, err, "teardown failed: err removing test stderr")
	err = os.Remove(cmd.stdout.Name())
	assert.NoErrorf(t, err, "teardown failed: err removing test stdout")
}

func indexWorkspaceChange(t *testing.T) (*commands, command) {
	cmds, cmd := tearUp(t)

	err := os.MkdirAll(filepath.Join(cmd.repo.Path, "a", "b"), 0755)
	assert.NoError(t, err)

	f, err := os.Create(filepath.Join(cmd.repo.Path, "1.txt"))
	assert.NoError(t, err)
	_, err = f.WriteString("one")
	assert.NoError(t, err)
	f.Close()

	f, err = os.Create(filepath.Join(cmd.repo.Path, "a", "2.txt"))
	assert.NoError(t, err)
	_, err = f.WriteString("two")
	assert.NoError(t, err)
	f.Close()

	f, err = os.Create(filepath.Join(cmd.repo.Path, "a", "b", "3.txt"))
	assert.NoError(t, err)
	_, err = f.WriteString("three")
	assert.NoError(t, err)
	f.Close()

	cmd.name = "add"
	cmd.args = []string{"."}

	exitCode, err := cmds.run(cmd)
	assert.NoErrorf(t, err, "error running `add` command")
	assert.Equal(t, 0, exitCode)

	cmd.name = "commit"
	cmd.args = []string{}
	cmd.stdin.WriteString("commit message")

	exitCode, err = cmds.run(cmd)
	assert.NoErrorf(t, err, "error running `add` command")
	assert.Equal(t, 0, exitCode)

	return cmds, cmd
}

// ---------------  Test functions  -------------------

func TestRepoInitialization(t *testing.T) {
	cmds, td := tearUpWithoutRepo(t)

	cmd := command{
		name:   "init",
		args:   []string{},
		env:    GetGitgoVar(),
		pwd:    td.dir,
		stdin:  td.stdin,
		stdout: td.stdout,
		stderr: td.stderr,
		repo:   gitgo.NewRepository(td.dir),
	}

	exitCode, err := cmds.run(cmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	dirInfos, err := os.ReadDir(filepath.Join(cmd.pwd, ".gitgo"))
	assert.NoErrorf(t, err, "error reading .gitgo dir")

	for _, dirInfo := range dirInfos {
		assert.Contains(t, gitgoFolders, dirInfo.Name())
	}

	tearDown(t, cmd)
}

func TestSingleFileAdd(t *testing.T) {
	cmds, cmd := tearUp(t)

	// If initialization of test repo is successful,
	// update the command name and related variables
	cmd.name = "add"
	cmd.args = []string{"file1.txt"}

	// Create the `file1.txt` in the test repo
	f, err := os.Create(filepath.Join(cmd.pwd, "file1.txt"))
	assert.NoError(t, err, "Error creating file in temp dir")
	assert.NotNilf(t, f, "Got nil, expected `*os.File`")

	// Write dummy data
	sentence := gofakeit.Sentence(5)
	_, err = f.WriteString(sentence)
	assert.NoError(t, err)

	// Run the `add` command
	exitCode, err := cmds.run(cmd)

	// Read standard output and standard error content
	cmd.stdout.Seek(0, 0)
	stdoutContent, _ := io.ReadAll(cmd.stdout)
	cmd.stderr.Seek(0, 0)
	stderrContent, _ := io.ReadAll(cmd.stderr)

	// Print content if test fails
	if err != nil || exitCode != 0 {
		t.Logf("STDOUT: %s", stdoutContent)
		t.Logf("STDERR: %s", stderrContent)
	}

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	objInfos, err := os.ReadDir(filepath.Join(cmd.pwd, ".gitgo", "objects"))
	assert.NoError(t, err, "error reading object dir")
	assert.Equal(t, 1, len(objInfos), "should have 1 blob file")

	tearDown(t, cmd)
}

func TestMultipleFileAdd(t *testing.T) {
	cmds, cmd := tearUp(t)

	cmd.name = "add"
	cmd.args = []string{"file1.txt", "file2.txt", "file3.txt"}

	// create files
	f1, err := os.Create(filepath.Join(cmd.pwd, "file1.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")
	assert.NotNilf(t, f1, "Got nil, expected `*os.File`")

	f2, err := os.Create(filepath.Join(cmd.pwd, "file2.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")
	assert.NotNilf(t, f2, "Got nil, expected `*os.File`")

	f3, err := os.Create(filepath.Join(cmd.pwd, "file3.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")
	assert.NotNilf(t, f3, "Got nil, expected `*os.File`")

	// Write dummy data
	sen := gofakeit.Sentence(6)
	_, err = f1.WriteString(sen)
	assert.NoErrorf(t, err, "Error writing dummy data")

	sen = gofakeit.Sentence(6)
	_, err = f2.WriteString(sen)
	assert.NoErrorf(t, err, "Error writing dummy data")

	sen = gofakeit.Sentence(6)
	_, err = f3.WriteString(sen)
	assert.NoErrorf(t, err, "Error writing dummy data")

	// Run the `add` command
	exitCode, err := cmds.run(cmd)

	// Read standard output and standard error content
	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)
	stderrCon, _ := io.ReadAll(cmd.stderr)

	// Print the std file content if test fails
	if err != nil || exitCode != 0 {
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	objInfos, err := os.ReadDir(filepath.Join(cmd.pwd, ".gitgo", "objects"))
	assert.NoErrorf(t, err, "error reading test repo objects dir")
	assert.Equalf(t, 3, len(objInfos), "Should have 3 blob files")

	tearDown(t, cmd)
}

func TestExecutableFileAdd(t *testing.T) {
	cmds, cmd := tearUp(t)

	cmd.name = "add"
	cmd.args = []string{"bankai.sh"}

	f, err := os.Create(filepath.Join(cmd.pwd, "bankai.sh"))
	assert.NoErrorf(t, err, "Error creating file in temp dir")
	assert.NotNilf(t, f, "Got nil, expected `*os.File`")

	sen := gofakeit.Sentence(3)
	_, err = f.WriteString(sen)
	assert.NoError(t, err)

	// Make the file into executable file, by the creator
	// (of course?)
	err = os.Chmod(f.Name(), 0755)
	assert.NoErrorf(t, err, "Error changing mod of test file")

	// run the `add` command
	exitCode, err := cmds.run(cmd)

	// Read std files
	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stderrCon, _ := io.ReadAll(cmd.stderr)
	stdoutCon, _ := io.ReadAll(cmd.stdout)

	assert.NoErrorf(t, err, fmt.Sprintf("%v", err))
	assert.Equalf(t, 0, exitCode, fmt.Sprintf("Stderr: %s\nStdout: %s\n", stderrCon, stdoutCon))

	objInfos, err := os.ReadDir(filepath.Join(cmd.pwd, ".gitgo", "objects"))
	assert.NoErrorf(t, err, "error reading test object dir")
	assert.Equalf(t, 1, len(objInfos), "should have 1 blob file")

	tearDown(t, cmd)
}

func TestIncrementalFileAdd(t *testing.T) {
	cmds, cmd := tearUp(t)

	cmd.name = "add"
	cmd.args = []string{"file1.txt"}

	// file 1
	f, err := os.Create(filepath.Join(cmd.pwd, "file1.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")
	assert.NotNilf(t, f, "Got nil, expected `*os.File`")

	sen := gofakeit.Sentence(5)
	_, err = f.WriteString(sen)
	assert.NoErrorf(t, err, "Error writing dummy data")

	exitCode, err := cmds.run(cmd)

	// Read std file data
	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)
	stderrCon, _ := io.ReadAll(cmd.stderr)

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	if err != nil || exitCode != 0 {
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	objInfos, err := os.ReadDir(filepath.Join(cmd.pwd, ".gitgo", "objects"))
	assert.NoErrorf(t, err, "error reading test repo objects dir")
	assert.Equalf(t, 1, len(objInfos), "Should have 1 blob files")

	// file 2

	f, err = os.Create(filepath.Join(cmd.pwd, "file2.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")
	assert.NotNilf(t, f, "Got nil, expected `*os.File`")

	sen = gofakeit.Sentence(3)
	_, err = f.WriteString(sen)
	assert.NoErrorf(t, err, "Error writing dummy data")

	cmd.args = []string{"file2.txt"}
	exitCode, err = cmds.run(cmd)

	// Read standard output and standard error content
	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ = io.ReadAll(cmd.stdout)
	stderrCon, _ = io.ReadAll(cmd.stderr)

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	// Print the std file content if test fails
	if err != nil || exitCode != 0 {
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	// file 3
	f, err = os.Create(filepath.Join(cmd.pwd, "file3.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")
	assert.NotNilf(t, f, "Got nil, expected `*os.File`")

	sen = gofakeit.Sentence(3)
	_, err = f.WriteString(sen)
	assert.NoErrorf(t, err, "Error writing dummy data")

	cmd.args = []string{"file3.txt"}
	exitCode, err = cmds.run(cmd)

	// Read standard output and standard error content
	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ = io.ReadAll(cmd.stdout)
	stderrCon, _ = io.ReadAll(cmd.stderr)

	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	// Print the std file content if test fails
	if err != nil || exitCode != 0 {
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}
}

func TestStatusCommand(t *testing.T) {
	t.Run("Basic Status command test", func(t *testing.T) {
		statusCommand(t)
	})
	t.Run("list files not tracked", func(t *testing.T) {
		testListFileAsUntrackedNotInIndex(t)
	})
	t.Run("lists untracked directories, not their contents", func(t *testing.T) {
		testUntrackedDir(t)
	})
	t.Run("list untracked files inside tracked directories", func(t *testing.T) {
		testUntrackedFileInTrackedDir(t)
	})
	t.Run("does not list empty untracked directories", func(t *testing.T) {
		testEmptyUntrackedDirectories(t)
	})
	t.Run("lists untracked directories that indirectly constain files", func(t *testing.T) {
		testIndirectlyUntrackedDirectories(t)
	})

	// Test cases for Index/Workspace difference
	t.Run("prints nothing when no files are changed", func(t *testing.T) {
		testPrintNothing(t)
	})
	t.Run("reports file with modified contents", func(t *testing.T) {
		testReportModifiedFiles(t)
	})
	t.Run("reports file with mode changed", func(t *testing.T) {
		testReportModeChanged(t)
	})
	t.Run("reports modified files without unchanged size", func(t *testing.T) {
		testReportModifiedWithUnchangedSize(t)
	})
	t.Run("prints nothing if file is touched", func(t *testing.T) {
		testReportNothingOnFileTouched(t)
	})
	t.Run("reports deleted file", func(t *testing.T) {
		testReportDeleteFile(t)
	})
	t.Run("reports files in a deleted directories", func(t *testing.T) {
		testReportDeleteFilesInDir(t)
	})
}

func statusCommand(t *testing.T) {
	cmds, cmd := tearUp(t)

	cmd.name = "status"
	cmd.args = []string{}

	_, err := os.Create(filepath.Join(cmd.pwd, "file1.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")

	_, err = os.Create(filepath.Join(cmd.pwd, "another.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")

	exitCode, err := cmds.run(cmd)

	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)
	stderrCon, _ := io.ReadAll(cmd.stderr)

	assert.NoErrorf(t, err, "Error running `status` command")
	assert.Equal(t, 0, exitCode)

	if err != nil || exitCode != 0 {
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	assert.True(t, strings.Contains(string(stdoutCon), "?? another.txt\n?? file1.txt\n"))
	// t.Log(string(stdoutCon))
}

func testListFileAsUntrackedNotInIndex(t *testing.T) {
	cmds, cmd := indexWorkspaceChange(t)

	_, err := os.Create(filepath.Join(cmd.pwd, "file.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")

	cmd.name = "status"
	cmd.stdout = tempFile("stdout")
	cmd.stderr = tempFile("stderr")

	exitCode, err := cmds.run(cmd)

	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)
	stderrCon, _ := io.ReadAll(cmd.stderr)

	assert.NoErrorf(t, err, "Error running `status` command")
	assert.Equal(t, 0, exitCode)

	if err != nil || exitCode != 0 {
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	assert.Equalf(t, "?? file.txt\n", string(stdoutCon), "Expected file not present in the output")
	// assert.Falsef(
	// 	t,
	// 	strings.Contains(string(stdoutCon), "?? committed.txt"),
	// 	"This file should not be in the output",
	// )
	// t.Log(string(stdoutCon))

	tearDown(t, cmd)
}

func testUntrackedDir(t *testing.T) {
	cmds, cmd := tearUp(t)

	_, err := os.Create(filepath.Join(cmd.pwd, "file.txt"))
	assert.NoErrorf(t, err, "Error creating file in test dir")
	err = os.Mkdir(filepath.Join(cmd.pwd, "dir"), 0755)
	assert.NoErrorf(t, err, "Error creating file and dir at the same time in test dir")
	_, err = os.Create(filepath.Join(cmd.pwd, "dir", "another.txt"))
	assert.NoError(t, err)

	cmd.name = "status"
	cmd.args = []string{}

	exitCode, err := cmds.run(cmd)

	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)
	stderrCon, _ := io.ReadAll(cmd.stderr)

	assert.NoErrorf(t, err, "Error running `status` command")
	assert.Equalf(t, 0, exitCode, "Not expected exit code")

	if err != nil || exitCode != 0 {
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	assert.Truef(
		t,
		strings.Contains(string(stdoutCon), "?? dir/\n?? file.txt"),
		"Expected dir & file not present in the output",
	)

	tearDown(t, cmd)
}

func testUntrackedFileInTrackedDir(t *testing.T) {
	cmds, cmd := tearUp(t)

	err := os.MkdirAll(filepath.Join(cmd.pwd, "a", "b"), 0755)
	assert.NoError(t, err)

	_, err = os.Create(filepath.Join(cmd.pwd, "a", "b", "inner.txt"))
	assert.NoError(t, err)

	cmd.name = "add"
	cmd.args = []string{"."}

	exitCode, err := cmds.run(cmd)

	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)
	stderrCon, _ := io.ReadAll(cmd.stderr)

	assert.NoErrorf(t, err, "Error running `status` command")
	assert.Equalf(t, 0, exitCode, "Not expected exit code")

	if err != nil || exitCode != 0 {
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	cmd.name = "commit"
	cmd.args = []string{}
	cmd.stdin.WriteString("commit message")

	exitCode, err = cmds.run(cmd)

	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ = io.ReadAll(cmd.stdout)
	stderrCon, _ = io.ReadAll(cmd.stderr)

	assert.NoErrorf(t, err, "Error running `commit` command")
	assert.Equalf(t, 0, exitCode, "Not expected exit code")

	if err != nil || exitCode != 0 {
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	_, err = os.Create(filepath.Join(cmd.pwd, "a", "outer.txt"))
	assert.NoError(t, err)

	err = os.MkdirAll(filepath.Join(cmd.pwd, "a", "b", "c"), 0755)
	assert.NoError(t, err)

	_, err = os.Create(filepath.Join(cmd.pwd, "a", "b", "c", "file.txt"))
	assert.NoError(t, err)

	cmd.name = "status"

	exitCode, err = cmds.run(cmd)

	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ = io.ReadAll(cmd.stdout)
	stderrCon, _ = io.ReadAll(cmd.stderr)

	assert.NoErrorf(t, err, "Error running `status` command")
	assert.Equalf(t, 0, exitCode, "Not expected exit code")

	if err != nil || exitCode != 0 {
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	assert.True(t, strings.Contains(
		string(stdoutCon),
		"?? a/b/c/\n?? a/outer.txt",
	))

	tearDown(t, cmd)
}

func testEmptyUntrackedDirectories(t *testing.T) {
	cmds, cmd := tearUp(t)

	err := os.Mkdir(filepath.Join(cmd.pwd, "outer"), 0755)
	assert.NoError(t, err)

	cmd.name = "status"
	cmd.args = []string{}
	cmd.stdout = tempFile("stdin")

	exitCode, err := cmds.run(cmd)

	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)
	stderrCon, _ := io.ReadAll(cmd.stderr)

	assert.NoErrorf(t, err, "Error running `status` command")
	assert.Equalf(t, 0, exitCode, "Not expected exit code")

	if err != nil || exitCode != 0 {
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	assert.False(t, strings.Contains(
		string(stdoutCon),
		"?? outer/",
	))

	tearDown(t, cmd)
}

func testIndirectlyUntrackedDirectories(t *testing.T) {
	cmds, cmd := tearUp(t)

	err := os.MkdirAll(filepath.Join(cmd.pwd, "outer", "inner"), 0755)
	assert.NoError(t, err)

	_, err = os.Create(filepath.Join(cmd.pwd, "outer", "inner", "file.txt"))
	assert.NoError(t, err)

	cmd.name = "status"
	cmd.args = []string{}

	exitCode, err := cmds.run(cmd)

	cmd.stdout.Seek(0, 0)
	cmd.stderr.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)
	stderrCon, _ := io.ReadAll(cmd.stderr)

	assert.NoErrorf(t, err, "Error running `status` command")
	assert.Equalf(t, 0, exitCode, "Not expected exit code")

	if err != nil || exitCode != 0 {
		if err != nil {
			t.Logf("ERROR: %v", err)
		}
		t.Logf("STDOUT: %s", stdoutCon)
		t.Logf("STDERR: %s", stderrCon)
	}

	assert.True(t, strings.Contains(
		string(stdoutCon),
		"?? outer/",
	))

	tearDown(t, cmd)
}

func testPrintNothing(t *testing.T) {
	cmds, cmd := indexWorkspaceChange(t)

	cmd.name = "status"
	cmd.args = []string{}
	cmd.stdout = tempFile("stdout")

	exitCode, err := cmds.run(cmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	cmd.stdout.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)

	assert.Equal(t, "", string(stdoutCon))

	tearDown(t, cmd)
}

func testReportModifiedFiles(t *testing.T) {
	cmds, cmd := indexWorkspaceChange(t)

	f, err := os.OpenFile(filepath.Join(cmd.repo.Path, "1.txt"), os.O_WRONLY, 0655)
	assert.NoError(t, err)
	_, err = f.WriteString("changed")
	assert.NoError(t, err)
	f.Close()

	f, err = os.OpenFile(filepath.Join(cmd.repo.Path, "a", "2.txt"), os.O_WRONLY, 0655)
	assert.NoError(t, err)
	_, err = f.WriteString("modified")
	assert.NoError(t, err)
	f.Close()

	cmd.name = "status"
	cmd.args = []string{}
	cmd.stdout = tempFile("stdout")

	code, err := cmds.run(cmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)

	cmd.stdout.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)

	assert.True(t, strings.Contains(string(stdoutCon), " M 1.txt\n M a/2.txt"))

	tearDown(t, cmd)
}

func testReportModeChanged(t *testing.T) {
	cmds, cmd := indexWorkspaceChange(t)

	err := os.Chmod(filepath.Join(cmd.repo.Path, "1.txt"), 0755)
	assert.NoError(t, err)

	cmd.name = "status"
	cmd.args = []string{}
	cmd.stdout = tempFile("stdout")

	code, err := cmds.run(cmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)

	cmd.stdout.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)

	assert.True(t, strings.Contains(string(stdoutCon), " M 1.txt"))

	tearDown(t, cmd)
}

func testReportModifiedWithUnchangedSize(t *testing.T) {
	cmds, cmd := indexWorkspaceChange(t)

	f, err := os.OpenFile(filepath.Join(cmd.repo.Path, "a", "b", "3.txt"), os.O_WRONLY, 0655)
	assert.NoError(t, err)
	assert.NotNil(t, f)
	_, err = f.WriteString("meoww")
	assert.NoError(t, err)
	f.Close()

	cmd.name = "status"
	cmd.args = []string{}
	cmd.stdout = tempFile("stdout")

	code, err := cmds.run(cmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)

	cmd.stdout.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)

	assert.Equal(t, " M a/b/3.txt\n", string(stdoutCon))

	tearDown(t, cmd)
}

func testReportNothingOnFileTouched(t *testing.T) {
	cmds, cmd := indexWorkspaceChange(t)

	exec.Command("touch", filepath.Join(cmd.repo.Path, "1.txt"))

	cmd.name = "status"
	cmd.args = []string{}
	cmd.stdout = tempFile("stdout")

	code, err := cmds.run(cmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)

	cmd.stdout.Seek(0, 0)
	stdoutCon, _ := io.ReadAll(cmd.stdout)

	assert.Equal(t, string(stdoutCon), "")

	tearDown(t, cmd)
}

func testReportDeleteFile(t *testing.T) {
	cmds, cmd := indexWorkspaceChange(t)

	err := os.Remove(filepath.Join(cmd.repo.Path, "a", "2.txt"))
	assert.NoError(t, err)

	cmd.name = "status"
	cmd.args = []string{}
	cmd.stdout = tempFile("stdout")

	code, err := cmds.run(cmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)

	cmd.stdout.Seek(0, 0)
	stdoutCon, err := io.ReadAll(cmd.stdout)

	assert.Equal(t, " D a/2.txt\n", string(stdoutCon))

	tearDown(t, cmd)
}

func testReportDeleteFilesInDir(t *testing.T) {
	cmds, cmd := indexWorkspaceChange(t)

	err := os.RemoveAll(filepath.Join(cmd.repo.Path, "a"))
	assert.NoError(t, err)

	cmd.name = "status"
	cmd.args = []string{}
	cmd.stdout = tempFile("stdout")

	code, err := cmds.run(cmd)
	assert.NoError(t, err)
	assert.Equal(t, 0, code)

	cmd.stdout.Seek(0, 0)
	stdoutCon, err := io.ReadAll(cmd.stdout)

	assert.Equal(t, " D a/2.txt\n D a/b/3.txt\n", string(stdoutCon))

	tearDown(t, cmd)
}
