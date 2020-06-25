<div align="center">
  <br>
  <img src="static/images/logo.png" width="256" alt="">
  <h1>PEPIC</h1>
</div>

PEPIC is a small proxy for uploading and serving pictures and videos. 
I use it for my [pet-projects](https://github.com/vas3k/vas3k.club) and on [my blog](https://vas3k.com). 
PEPIC can convert, resize and optimize media files in-flight to save you monies and bandwidth. And it's dead simple.

It's not meant to be used by anyone else, but if you're brave enough — give it a try. Maybe we'll become friends.

## 🔮 Building and running locally

1. Get [Docker](https://www.docker.com/get-started)

2. Clone the repo

```
git clone git@github.com:vas3k/pepic.git
cd pepic
```

3. Build and run the app

```
docker build .
docker run -p 8118:8118 -v ${PWD}/uploads:/app/uploads $(docker build -q .)
```

4. Go to [http://localhost:8118](http://localhost:8118) and try uploading some stuff. 
Check the data directory (`./uploads`) after that. It should have some files.

5. Try to resize an image by adding a number of pixels to its URL. For example: `https://localhost:8118/file.jpg -> https://localhost:8118/500/file.jpg`

6. Check out the [config/config.yml](config/config.yml) file. Some stuff is turned off by default.
You can tweak them for yourself and rebuild the docker again (step 3) to apply them.

![](static/images/screenshot1.png)

## 🚢 Production Usage

> ⚠️ If you plan to host anything bigger than a blog, put it behind caching CDN. 
> CloudFlare offers a free one if you don't hate big corporations :D

Let's say, you want to host it on `https://media.mydomain.org`

#### 1. Modify `config/config.yml` to your taste

```
global:
  host: 0.0.0.0 
  port: 8118  # internal host and port, leave it as it is
  base_url: "https://media.mydomain.org"
  secret_code: "secretpass"
  max_upload_size: "500M"
```

#### 2. Build and run production docker

Don't forget to mount upload volume to store files on host (or you can lose those files if container will be killed).

```
docker run -p 8118:8118 -v /host/dir/uploads:/app/uploads $(docker build -q .)
```

If you prefer docker-compose, you can use it too. Check out the included [docker-compose.example.yml](docker-compose.example.yml). 
You can easily transform it into your favourite k8s config or whatever is fashionable this summer. 

> 👍 Don't forger to periodically backup the `/host/dir/uploads` directory just in case :)

#### 3. Use nginx or your other favourite proxy

Just proxy all calls from the domain (media.mydomain.org) to pepic backend (0.0.0.0:8118). It can handle static files too.

```
server {
    listen 80;
    server_name media.mydomain.org;

    location / {
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Host $http_host;
        proxy_pass http://0.0.0.0:8118;
    }
}
```

## 😍 Contributions

Contributions are welcome.  

Open an [Issue](https://github.com/vas3k/vas3k.club/issues) if you want to report a bug and propose an idea.

## ✅ TODO

- [ ] Convert/quality API flag
- [ ] Proper Accept header check + JSON upload and API
- [ ] Tests :D

## 👩‍💼 License 

It's [MIT](LICENSE).

Contact me if you have any questions — [me@vas3k.ru](mailto:me@vas3k.ru).

❤️
