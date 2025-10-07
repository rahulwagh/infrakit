1. Install cobra - 

```
go get -u github.com/spf13/cobra@latest

go install github.com/spf13/cobra-cli@latest
```

2. Set cobra-cli to system path

```
nano ~/.zshrc

```

Add following line into the `.zshrc`

```
export PATH=$PATH:/Users/rahulwagh/go/bin
```

3. initialize it 

```
cobra init --author "Rahul Wagh rahul.wagh@jhooq.com"
```

4. Add Your sync and search Commands:
```
cobra-cli add sync
cobra-cli add search

```

5. The cache.json file is stored in a hidden directory named .cloudgrep inside your user's home directory.

The exact path depends on your operating system:

   - On macOS or Linux: ~/.infrakit/cache.json (which is the same as /Users/your-username/.cloudgrep/cache.json)

   - On Windows: C:\Users\your-username\.infrakit\cache.json


 6.The Fuzzy Searcher cli -

 ```
 go get github.com/ktr0731/go-fuzzyfinder
 ```

 7. Install a non-interactive fuzzy search library:
 ```
 go get github.com/lithammer/fuzzysearch
 ```

 8. Use the Cobra generator to add the serve command:

 ```
 cobra-cli add serve
 ```