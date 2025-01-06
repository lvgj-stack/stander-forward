#!/bin/bash

# 变量定义
CONTROLLER_ADDR=$1
NODE_KEY=$2
EXTRA_ARGS=$3
GITHUB_REPO="https://github.com/lvgj-stack/naive-admin-go/releases/download/v1.0.0/stander"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="stander-agent"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"

# 检查是否以 root 用户运行
if [ "$EUID" -ne 0 ]; then
    echo "请以 root 用户运行该脚本"
    exit 1
fi
# 检查参数是否为空
if [ -z "$CONTROLLER_ADDR" ]; then
    echo "错误: CONTROLLER_ADDR 参数不能为空"
    exit 1
fi

if [ -z "$NODE_KEY" ]; then
    echo "错误: NODE_KEY 参数不能为空"
    exit 1
fi

# 从 GitHub 拉取二进制文件
echo "正在下载二进制文件..."
if [ -f "$INSTALL_DIR/stander-agent" ]; then
  echo "文件存在，跳过下载..."
else
  curl -L -o "$INSTALL_DIR/stander-agent" "$GITHUB_REPO"
fi

# 检查下载是否成功
if [ $? -ne 0 ]; then
    echo "下载失败，请检查 GitHub 地址"
    exit 1
fi

# 赋予执行权限
chmod +x "$INSTALL_DIR/stander-agent"

# 创建 systemd 服务文件
echo "正在创建 systemd 服务..."
cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=Stander转发Agent端
ConditionFileIsExecutable=/usr/local/bin/stander-agent


[Service]
StartLimitInterval=5
StartLimitBurst=10
ExecStart=$INSTALL_DIR/stander-agent -a $CONTROLLER_ADDR -k $NODE_KEY $EXTRA_ARGS
WorkingDirectory=/root

Restart=always

RestartSec=120

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