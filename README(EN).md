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
sudo yum install php php-pdo php-mysql httpd mysql55-server memcached php-pecl-memcache php-mbstring php-pecl-imagick git
```

### 3. Deploy codes

```BASH
git clone https://github.com/KLab/emlauncher.git
cd emlauncher
git submodule init
git submodule update
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
sudo /etc/init.d/httpd start
sudo chkconfig httpd on
```


### 5. Database setup

```BASH
sudo /etc/init.d/mysqld start
sudo chkconfig mysqld on
```

Create file with username and password of DB.

Eg:
```
echo 'emlauncher:password' > $HOME/dbauth
```

Modify the passwords of data/sql/database.sql, and send to MySQL.
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
Copy ``config/mfw_serverenv_config_sample.php``,and rewrite
``$serverenv_config['ec2']['database']['authfile']`` to the path of dbauth file that was created at 5.

#### emlauncher_config.php
Copy ``config/emlauncher_config_sample.php``, and rewrite to match your own environment.

Create bucket that will be assigned to bucket name of S3 in advance. 

### 8. Complete

Access instance with HTTP in browser.
When the login page of EMLauncher is displayed, it’s complete.

