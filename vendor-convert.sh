cat Godeps.json | jq -c -r '.[] | "\(.ImportPath) \(.Rev)"'
