Develop EMLauncher on docker container
---

# How to run

1. Checkout submodules
```sh
git submodule init
git submodule update
```

2. Modify configs
```sh
cp config/{emlauncher_config_sample.php,emlauncher_config.php}
cp config/{mfw_serverenv_config_docker.php,mfw_serverenv_config.php}

# set your aws keys and s3 bucket name.
vim config/emlauncher_config.php
```

3. Build and run docker
```sh
docker-compose up --build
```

4. Add EMLauncher user
```sh
docker exec $(docker ps -f "name=emlauncher_db" -q) mysql -uroot -ppassword emlauncher -e "INSERT INTO user_pass (mail) VALUES ('your-name@example.com');"
```

5. Open EMLauncher in a browser
http://localhost:10080

