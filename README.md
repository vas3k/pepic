<div align="center">
  <br>
  <img src="static/images/logo.png" width="256" alt="">
  <h1>PEPIC</h1>
</div>

Pepic is a small self-hosted media proxy that helps me to upload, store, serve and convert pictures and videos on my own servers locally.

Currently, I use it as a main storage for media files in my [pet-projects](https://github.com/vas3k/vas3k.club) and on [my blog](https://vas3k.blog).

Pepic can upload and optimize media files in-flight to save you money and bandwidth. It's highly recommended to use it in combination with Cloudflare CDN for better caching.

Internally, Pepic it uses [ffmpeg](https://ffmpeg.org/download.html) for videos and [vips](https://libvips.github.io/libvips/install.html) for images, which makes it quite fast and supports many media file formats.

Images: **JPG, PNG, GIF, WEBP, SVG, HEIF, TIFF, AVIF, etc**
Video: **basically everything ffmpeg supports**

Pepic is open source, however it's not meant to be used by anyone. Only if you're brave. Scroll down this README for better alternatives.


## Main Features

- **Local files upload**: Accept files in multipart/form-data or bytes and store them to a local directory.
- **Automatic GIF to video conversion**: Convert GIFs to videos because GIFs suck, slow down web pages, and don't support hardware acceleration.
- **Media format conversion and optimization**: Convert and optimize media files on-the-fly.
- **Dynamic resizing**: Easily resize images in real-time by modifying the URL, which helps in reducing bandwidth and storage space on devices.
- **High performance**: Pepic uses native libraries like `ffmpeg` and `vips` for video and image processing to ensure high performance and fast processing times.
- **Local and containerized environments**: Designed to run smoothly in both local environments and within Docker containers, making it versatile for development and deployment.
- **Custom configuration**: Flexible configuration options through `config.yml`, allowing adjustments to image size, quality, automatic conversion, templates, etc.

![](static/images/screenshot1.png)

## 🤖 How to Run

1. Install `vips` and `ffmpeg` first, as they are external dependencies.

```bash
brew install vips ffmpeg
```

2. Use the following command to start a local server on [localhost:8118](http://localhost:8118).

```bash
go run main.go serve --config ./etc/pepic/config.yml
```

> ⚠️ If you're getting `invalid flag in pkg-config` error, run `brew install pkg-config` and `export CGO_CFLAGS_ALLOW="-Xpreprocessor"`. Then try `go run` again.

3. Enjoy!

## 🐳 Running in Docker

1. Get [Docker](https://www.docker.com/get-started)

2. Clone the repo

```bash
git clone git@github.com:vas3k/pepic.git
cd pepic
```

3. Build and run the app

```bash
docker build .
docker run -p 8118:8118 -v ${PWD}/uploads:/app/uploads $(docker build -q .)
```

4. Go to [http://localhost:8118](http://localhost:8118) and try uploading something. You should see uploaded images or videos in the data directory (`./uploads`) after that.

5. Try to resize an image by adding a number of pixels to its URL. For example: `https://localhost:8118/file.jpg -> https://localhost:8118/500/file.jpg`

6. Check out the [etc/pepic/config.yml](etc/pepic/config.yml) file. Some stuff is turned off by default. You can tweak them for yourself and rebuild the Docker again (step 3) to apply them.


## 🚢 Production Deployment

> ⚠️ If you plan to host anything bigger than a blog, always put Pepic behind a CDN. CloudFlare offers a free one if you don't hate big corporations :D

Let's say, you want to host it on `https://media.mydomain.org`

1. Modify `etc/pepic/config.yml` to your taste

```yaml
global:
  host: 0.0.0.0 
  port: 8118  # internal host and port, leave it as it is
  base_url: "https://media.mydomain.org"
  secret_code: "secretpass"
  max_upload_size: "500M"
```

2. Build and run production Docker

Don't forget to mount upload volume to store files on host (or you can lose those files when the container is killed).

```bash
docker run -p 8118:8118 -v /host/dir/uploads:/app/uploads --restart=always $(docker build -q .)
```

If you prefer docker-compose, you can use it too. Check out the included [docker-compose.example.yml](docker-compose.example.yml). 
You can easily transform it into your favourite k8s config or whatever is fashionable this summer. 

> 👍 Don't forget to periodically backup the `/host/dir/uploads` directory just in case :)

3. Use nginx or your other favourite proxy

Just proxy all calls from the domain (media.mydomain.org) to Pepic backend (0.0.0.0:8118). Don't forget to set `max file size` and `proxy timeot` directives to avoid gateway errors on big files (especially videos).

```nginx
server {
    listen 80;
    server_name media.mydomain.org;

    client_max_body_size 500M;
    real_ip_header X-Real-IP;

    location / {
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 300;
        proxy_connect_timeout 300;
        proxy_send_timeout 300;
        send_timeout 300;

        proxy_set_header Host $http_host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect off;
        proxy_buffering off;

        proxy_pass http://0.0.0.0:8118;
    }
}
```

## 😍 Contributions

Contributions are welcome.  

Open an [Issue](https://github.com/vas3k/vas3k.club/issues) if you want to report a bug or propose an idea.

## ✅ TODO

- [ ] Tests :D
- [ ] Upload by URL
- [ ] Crop, rotate and other useful transformations (face blur? pre-loader generator?)
- [ ] Live conversion by changing file's extension 
- [ ] Set format and media quality during the upload (using GET/POST params?)


## 🤔 Alternatives

After reading all this, you probably realized how bad it is and looking for other alternatives. Here's my recommendations:

- [imgproxy](https://github.com/imgproxy/imgproxy)
- [imaginary](https://github.com/h2non/imaginary)
- [flyimg](https://github.com/flyimg/flyimg)


## 👩‍💼 License 

It's [MIT](LICENSE).

Contact me if you have any questions — [me@vas3k.ru](mailto:me@vas3k.ru).

❤️
