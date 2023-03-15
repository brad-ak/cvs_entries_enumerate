# /CVS/Entries Folder Enumerator

A simple go script that automatically enumerates an exposed `/CVS/Entries` folder.

The script checks a number of files for the given host, aggregates the file paths found in each file, checks for availability of the file paths, and downloads any files that were available. All files will be saved off in the directory structure listed in the found filepaths under the host name parent directory.

Supports proxy and running multiple threads.

## Usage

```
Usage of ./cvs_enum:
  -host string
    	Url to target. Example: https://example.com (default "NOHOST")
  -list string
    	List of hosts. Example: host_list.txt (default "NOLIST")
  -proxy string
    	Proxy host and port. Example: http://127.0.0.1:8080 (default "NOPROXY")
  -threads int
    	Number of concurrent threads to run. Example: 100 (default 20)
```

## Example

```
❯❯ ./cvs_enum --host https://fakehost.com
[+] Testing https://fakehost.com

[!] Found 30 filepaths. Attempting to download valid paths.
Downloaded file testfile.js with size 6324
Downloaded file testfile.html with size 48
Downloaded file testfile.phtml with size 0
Downloaded file testfile.php with size 0

[!] Directories found:
/testdir1
/testdir2
/testdir3
/testdir4

===============================
Done
```
