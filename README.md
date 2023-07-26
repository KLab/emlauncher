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
sudo amazon-linux-extras install php8.2
sudo amazon-linux-extras install mariadb10.5
sudo amazon-linux-extras install memcached1.5
sudo yum groupinstall "Development Tools"
sudo yum install httpd php php-gd php-mbstring php-xml git php-devel php-pear zlib-devel libmemcached.x86_64 libmemcached-devel.x86_64 ImageMagick.x86_64 ImageMagick-devel.x86_64 libzip-devel.x86_64
sudo pecl install igbinary
sudo pecl install msgpack
sudo pecl install memcached
sudo pecl install imagick
sudo sh -c "echo 'extension=imagick.so' > /etc/php.d/40-imagick.ini"
sudo sh -c 'cat <<EOF > /etc/php.d/50-memcached.ini
extension=memcached.so

[memcached]
memcached.sess_locking = On
memcached.sess_lock_wait_min = 1000
memcached.sess_lock_wait_max = 2000
memcached.sess_lock_retries = 5
memcached.sess_lock_expire = 0
memcached.sess_prefix = "memc.sess.key."
memcached.sess_persistent = Off
memcached.sess_consistent_hash = On
memcached.sess_remove_failed_servers = Off
memcached.sess_number_of_replicas = 0
memcached.sess_binary_protocol = On
memcached.sess_randomize_replica_read = Off
memcached.sess_connect_timeout = 1000
memcached.sess_sasl_username = NULL
memcached.sess_sasl_password = NULL
memcached.compression_type = "fastlz"
memcached.compression_factor = "1.3"
memcached.compression_threshold = 2000
memcached.serializer = "igbinary"
memcached.store_retry_count = 2
memcached.default_consistent_hash = Off
memcached.default_binary_protocol = Off
memcached.default_connect_timeout = 0
EOF'
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
sudo mysql -uroot < /path/to/emlauncher/data/sql/database.sql
sudo mysql -uroot emlauncher < /path/to/emlauncher/data/sql/tables.sql
```

### 6. Memcache setup

```BASH
sudo systemctl start memcached
sudo systemctl enable memcached
```

### 7. Setup bundletool for Android App Bundle

```BASH
sudo yum install java-1.8.0-openjdk-headless
curl -sLO https://github.com/google/bundletool/releases/download/1.15.1/bundletool-all-1.15.1.jar
```

APKを再署名するためのキーストアも用意します。
ここで設定するパスワード、キーストアファイル名、エイリアス名はこの後設定ファイル`emlauncher_config.php`に記載します。
```BASH
keytool -genkey -keystore {emlauncher-keystore.jks} -keyalg RSA -keysize 2048 -validity 10000 -alias {key-alias}
```

#### AARCH64(ARM64)アーキテクチャのプラットホームで動作させる場合
AARCH64(ARM64)アーキテクチャのプラットホームで動作させる場合にはbundletoolに内包のaapt2がAMD64(x86_64)アーキテクチャ向けでそのままでは動作しないのでARM64アーキテクチャー用のaapt2をgithubからダウンロードします。
```BASH
curl -sLO https://github.com/JonForShort/android-tools/raw/master/build/android-9.0.0_r33/aapt2/arm64-v8a/bin/aapt2
```

### 8. Configuration

#### mfw_serverevn_config.php
``config/mfw_serverenv_config_sample.php``をコピーし、``$serverenv_config['ec2']['database']['authfile']``を
5で作成したdbauthファイルのパスに書き換えます。

#### emlauncher_config.php
``config/emlauncher_config_sample.php``をコピーし、自身の環境に合わせて書き換えます。

S3のbucket名に指定するbucketは予め作成しておきます。

##### AARCH64(ARM64)アーキテクチャのプラットホームで動作させる場合
APKファイルの設定のaapt2に(6.)でダウンロードしたARM64向けaapt2実行ファイルのパスを指定します。

### 9. Complete

ブラウザでインスタンスにHTTPでアクセスします。
EMLauncherのログインページが表示されたら完了です。
