mailrender - Use the API to render emails as images or PDF
============

### Quick start
1. run `mailrender` server
```bash
docker run --rm -ti --shm-size=500m -p 8000:8000 restsend/mailrender
```
2. Render a .eml file:
```sh
curl -o output.png -X POST 'http://localhost:8000/mailrender?format=png&device=web' \
    -F "content=@example.eml"
```

### API
Submit the data request in the way of `POST`, parameters through `urlquery`:
1. Parameters of url query：
    - `format` output format: `png` and `pdf`
    - `device` Dimensions of the `device` page:
        - `web` Laptop screen size: 1440 x 900 high-resolution monitor
        - `iphone` iPhone Display: 736 x 414 Retina Display
    - `timezone` Show when the email was sent: Default is `America/New York`
    - `waitload` The seconds to wait for the page to load, the default is `60` seconds
    - `headless` Only render email content, default is `false`
    - `textonly` Only render text, if there is html will not render, the default is `false`
    - `author` Author information of picture or PDF
    - `watermark` Show watermark on image
1. Parameters for form-data：
    - `content` file type, *required*


### How mailrender work?
1. Parse the `eml` file, unpack the attachment and embed files, to ensure that the mail can be displayed normally
1. Render the email content through `chrome-headless` to get the image/pdf.

### Deploy Docker service
If Docker runs mailrender, you need to add `--shm-size=500m` to the run parameter of 
docker, for example:
```bash
docker run --rm -ti --shm-size=500m -p 8000:8000 restsend/mailrender
```

The following environment variables can be adjusted:
- `PORT` HTTP server port ,default: `8000`
- `SIZELIMIT` Upload file size limit(MB), default: `50` 
- `AUTHOR` Author info, default: `https://github.com/restsend/mailrender`
- `STORE` temporary storage directory, default: `/tmp/mailrender`

```bash
docker run --rm -ti --shm-size=500m -e PORT=8080 -p 8080:8080 restsend/mailrender
```


### Add or change font
We have built-in noto fonts, if you need to modify the fonts, add fonts in `fonts/`, and modify `conf/local.conf` to configure different font matching

### How to build
There are two ways to compile `mailrender`:
1. docker build .
1. go mod download && go build .

