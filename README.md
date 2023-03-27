mailrender - Use the API to render emails as images or PDF
============

### Quick start
1. run `mailrender` server
```bash
go run .
```
2. Render a .eml file:
```sh
curl -o output.png -X POST 'http://localhost:8000/mailrender?format=png&device=web' \
    -F "content=@example.eml"
```


### How mailrender work?

### Add or change font

### How to build
