#!/bin/bash

# 变量定义
GITHUB_REPO="https://file.byte.gs/gost"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="gost"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"

# 检查是否以 root 用户运行
if [ "$EUID" -ne 0 ]; then
    echo "请以 root 用户运行该脚本"
    exit 1
fi

# 从 GitHub 拉取二进制文件
echo "正在下载二进制文件..."
if [ -f "$INSTALL_DIR/gost" ]; then
  echo "文件存在，跳过下载..."
else
  curl -L -o "$INSTALL_DIR/gost" "$GITHUB_REPO"
fi

# 检查下载是否成功
if [ $? -ne 0 ]; then
    echo "下载失败，请检查 GitHub 地址"
    exit 1
fi

# 赋予执行权限
chmod +x "$INSTALL_DIR/gost"
dir="/etc/gost"
if [ ! -d "$dir" ]; then
    mkdir -p "$dir"
    echo "已创建目录 $dir"
else
    echo "目录 $dir 已存在，无需创建"
fi

cd $dir
wget -O certFile.pem https://file.byte.gs/certFile.pem
wget -O key.pem https://file.byte.gs/key.pem

# 创建 systemd 服务文件
echo "正在创建 systemd 服务..."
cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=Stander转发Agent端-Gost
ConditionFileIsExecutable=/usr/local/bin/gost


[Service]
StartLimitInterval=5
StartLimitBurst=10
Environment="GOST_LOGGER_LEVEL=warn"
ExecStart=$INSTALL_DIR/gost -api 'http://127.0.0.1:19123'
WorkingDirectory=/root
Restart=always

RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# 重新加载 systemd 配置
echo "正在重新加载 systemd 配置..."
systemctl daemon-reload

# 启用并启动服务
echo "正在启用并启动服务..."
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"

echo "安装完成！"
