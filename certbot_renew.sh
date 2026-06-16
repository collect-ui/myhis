#!/bin/bash

# 定义域名和邮箱
DOMAINS=("collect-ui.top")
EMAIL="18874848657@163.com"

# 生成证书
generate_certificate() {
    for DOMAIN in "${DOMAINS[@]}"; do
        echo "Generating SSL certificate for $DOMAIN..."
        sudo certbot certonly --standalone -d $DOMAIN --email $EMAIL --agree-tos --non-interactive
    done
}

# 自动续期证书
renew_certificate() {
    echo "Renewing SSL certificates..."
    sudo certbot renew --quiet
}

# 主函数
main() {
    # 检查是否已经存在证书
    for DOMAIN in "${DOMAINS[@]}"; do
        if [ ! -d "/etc/letsencrypt/live/$DOMAIN" ]; then
            generate_certificate
            break
        else
            echo "SSL certificate for $DOMAIN already exists."
        fi
    done

    # 设置自动续期
    echo "Setting up automatic renewal..."
    (crontab -l 2>/dev/null; echo "0 0 * * * /usr/bin/certbot renew --quiet") | sudo crontab -
    echo "Automatic renewal setup complete."
}

# 执行主函数
main