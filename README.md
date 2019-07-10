EMLauncher
==========

## Setup

EC2のAmazonLinux2でEMLauncherを動かす手順です。

### 1. Launch EC2 instance

インスタンスを立ち上げたらセキュリティグループの設定でHTTP(80)を許可しておきます。

t1.microの場合はメモリが足りなくなることがあるので、swapファイルを用意します。
```BASH
sudo dd if=/dev/zero of=/swapfile bs=1M count=1024
sudo mkswap /swapfile
sudo swapon /swapfile
sudo sh -c "echo '/swapfile swap swap defaults 0 0' >> /etc/fstab"
```

### 2. Install required packages

```BASH
sudo amazon-linux-extras install lamp-mariadb10.2-php7.2
sudo amazon-linux-extras install memcached1.5
sudo yum install mariadb-server httpd php-gd php-mbstring php-xml php-pecl-imagick php-pecl-memcached php-pecl-zip git
curl -sS https://getcomposer.org/installer | php
sudo cp composer.phar /usr/local/bin/composer
```

### 3. Deploy codes

```BASH
git clone https://github.com/KLab/emlauncher.git
cd emlauncher
git submodule init
git submodule update
composer install
```
Apacheがファイルにアクセスできるようにパーミッションを調整してください。

### 4. Apache setup

/etc/httpd/conf/httpd.confを編集します。
```XML
DocumentRoot "/path/to/emlauncher/web"
SetEnv MFW_ENV 'ec2'
<Directory "/path/to/emlauncher/web">
  AllowOverride All
  ...略...
</Directory>
```

```BASH
sudo systemctl start httpd
sudo systemctl enable httpd
```


### 5. Database setup

```BASH
sudo systemctl start mariadb
sudo systemctl enable mariadb
```

DBのユーザ名、パスワードを書いたファイルを作成します。

例:
```
echo 'emlauncher:password' > $HOME/dbauth
```

data/sql/database.sqlのパスワードを合わせて修正し、MySQLに流します。
```BASH
mysql -uroot < /path/to/emlauncher/data/sql/database.sql
mysql -uroot emlauncher < /path/to/emlauncher/data/sql/tables.sql
```

### 6. Memcache setup

```BASH
sudo systemctl start memcached
sudo systemctl enable memcached
```

### 7. Setup bundletool for Android App Bundle

```BASH
sudo yum install java-1.8.0-openjdk-headless
curl -sLO https://github.com/google/bundletool/releases/download/0.10.0/bundletool-all-0.10.0.jar
```

APKを再署名するためのキーストアも用意します。
ここで設定するパスワード、キーストアファイル名、エイリアス名はこの後設定ファイル`emlauncher_config.php`に記載します。
```BASH
keytool -genkey -keystore {emlauncher-keystore.jks} -keyalg RSA -keysize 2048 -validity 10000 -alias {key-alias}
```

### 8. Configuration

#### mfw_serverevn_config.php
``config/mfw_serverenv_config_sample.php``をコピーし、``$serverenv_config['ec2']['database']['authfile']``を
5で作成したdbauthファイルのパスに書き換えます。

#### emlauncher_config.php
``config/emlauncher_config_sample.php``をコピーし、自身の環境に合わせて書き換えます。

S3のbucket名に指定するbucketは予め作成しておきます。

### 9. Complete

ブラウザでインスタンスにHTTPでアクセスします。
EMLauncherのログインページが表示されたら完了です。

