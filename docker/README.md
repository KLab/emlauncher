Develop EMLauncher on docker containers
---

# How to run (without s3 mock)

1. Setup libraries
```sh
git submodule init
git submodule update
compose install
```

2. Modify configs
```sh
cp config/emlauncher_config{_sample,}.php
cp config/mfw_serverenv_config{_sample,}.php

# set your aws keys and s3 bucket name.
vim config/emlauncher_config.php
```

3. Build and run docker
```sh
docker-compose up --build
```

4. Add EMLauncher user
```sh
docker-compose exec db mysql -uroot -ppassword emlauncher -e 'INSERT INTO user_pass (mail) VALUES ("your-name@example.com");'
```

5. Open EMLauncher in a browser

http://localhost:8081

# How to run (with s3 mock)

1. Setup libraries
```sh
git submodule init
git submodule update
composer install
```

2. Modify configs
```sh
cp config/emlauncher_config{_sample,}.php
cp config/mfw_serverenv_config{_sample,}.php

# set your aws keys and s3 bucket name, **s3 mock url**, and change storage_class to 'S3'.
vim config/emlauncher_config.php
```

3. Build and run docker
```sh
ln -snvf docker-compose.s3-localstack.yml docker-compose.override.yml
docker-compose up --build
```

4. Add EMLauncher user
```sh
docker-compose exec db mysql -uroot -ppassword emlauncher -e 'INSERT INTO user_pass (mail) VALUES ("your-name@example.com");'
```

5. Create s3 bucket on s3 mock

```sh
aws --endpoint-url=http://localhost:4572 s3 mb s3://emlauncher-dev/
```

6. Open EMLauncher in a browser

http://localhost:8081

