# Run from the repository root with a current apibconv binary on PATH:
#   vhs docs/demo/demo.tape
Output docs/demo/apibconv.gif
Set Shell "bash"
Set FontSize 19
Set Width 1000
Set Height 690
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set TypingSpeed 30ms
Hide
Type "workdir=$(mktemp -d); cp docs/demo/api.apib $workdir/api.apib; cd $workdir; clear"
Enter
Sleep 1s
Show
Type "cat api.apib"
Enter
Sleep 3s
Type "clear"
Enter
Type "apibconv -o openapi.yaml api.apib"
Enter
Sleep 2s
Type "head -n 15 openapi.yaml"
Enter
Sleep 4s
Type "apibconv --validate openapi.yaml"
Enter
Sleep 4s
