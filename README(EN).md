EMLauncher
==========

## Setup

This is the process to run EMLauncher in AmazonLinux of EC2.

### 1. Launch EC2 instance

Once the instance is established, permit HTTP(80) in the security group setting.

In the case of t1.micro, there are cases of lack of memory, it is necessary to prepare swap file.
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
sudo sh -c "cat /etc/php.d/20-zip.ini | sed 's/zip/memcached/' > /etc/php.d/30-memcached.ini"
sudo sh -c "cat /etc/php.d/20-zip.ini | sed 's/zip/imagick/' > /etc/php.d/30-imagick.ini"
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
Please modify the permission so that Apache can access the file.

### 4. Apache setup

Edit /etc/httpd/conf/httpd.conf
```XML
DocumentRoot "/path/to/emlauncher/web"
SetEnv MFW_ENV 'ec2'
<Directory "/path/to/emlauncher/web">
  AllowOverride All
  ...abbreviated..
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

Create file with username and password of DB.

Eg:
```
echo 'emlauncher:password' > $HOME/dbauth
```

Modify the passwords of data/sql/database.sql, and send to MySQL.
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

Here will also have a keystore for re-signing the generate APK.
The password, keystore file name, and alias name to be set here will be described later in the configuration file `emlauncher_config.php`.
```BASH
keytool -genkey -keystore {emlauncher-keystore.jks} -keyalg RSA -keysize 2048 -validity 10000 -alias {key-alias}
```

#### When Running on a platform with AARCH64(ARM64) architecture
When Running on the platform of AARCH64(ARM64) architecture, aapt2 included in bundletool is for AMD64(x86_64) architecture and does not work as it is,
so download aapt2 for ARM64 architecture from github.
```BASH
curl -sLO https://github.com/JonForShort/android-tools/raw/master/build/android-9.0.0_r33/aapt2/arm64-v8a/bin/aapt2
```

### 8. Configuration

#### mfw_serverevn_config.php
Copy ``config/mfw_serverenv_config_sample.php``,and rewrite
``$serverenv_config['ec2']['database']['authfile']`` to the path of dbauth file that was created at 5.

#### emlauncher_config.php
Copy ``config/emlauncher_config_sample.php``, and rewrite to match your own environment.

Create bucket that will be assigned to bucket name of S3 in advance.

##### When running on a platform with AARCH64(ARM64) architecture
Specify the path of the aapt2 executable file for ARM64 downloaded at section (6.) in aapt2 of the APK file settings.

### 9. Complete

Access instance with HTTP in browser.
When the login page of EMLauncher is displayed, it’s complete.
