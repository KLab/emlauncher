EMLauncher
==========

## Setup

EC2のAmazonLinuxでEMLauncherを動かす手順です。

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
sudo yum install php php-pdo php-mysql httpd mysql55-server memcached php-pecl-memcache php-mbstring php-pecl-imagick git
```

### 3. Deploy codes

```BASH
git clone https://github.com/KLab/emlauncher.git
cd emlauncher
git submodule init
git submodule update
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
sudo /etc/init.d/httpd start
sudo chkconfig httpd on
```


### 5. Database setup

```BASH
sudo /etc/init.d/mysqld start
sudo chkconfig mysqld on
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
sudo /etc/init.d/memcached start
sudo chkconfig memcached on
```

### 7. Configuration

#### mfw_serverevn_config.php
``config/mfw_serverenv_config_sample.php``をコピーし、``$serverenv_config['ec2']['database']['authfile']``を
5で作成したdbauthファイルのパスに書き換えます。

#### emlauncher_config.php
``config/emlauncher_config_sample.php``をコピーし、自身の環境に合わせて書き換えます。

S3のbucket名に指定するbucketは予め作成しておきます。

### 8. Complete

ブラウザでインスタンスにHTTPでアクセスします。
EMLauncherのログインページが表示されたら完了です。

