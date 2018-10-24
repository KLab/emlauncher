#!/bin/sh
echo "rewriting hosts..."
line1=$(cat /etc/hosts | grep -v "localhost" | tail -n 1)
line2=$(echo $line1 | awk '{print $2}')
echo "$line1 $line2.localdomain" >> /etc/hosts

cat /etc/hosts

echo "restarting sendmail..."
service sendmail restart

echo "run docker-php-entrypoint"
/usr/local/bin/docker-php-entrypoint /usr/local/bin/apache2-foreground
