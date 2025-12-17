# 静态资源目录

## 目录说明

此目录用于存放 HTTP Server 的静态资源文件。

## 文件放置规则

### 1. 默认头像文件

**文件名**：`default_avatar.png`  
**路径**：`httpserver/static/default_avatar.png`  
**用途**：当用户未上传头像时，返回此默认头像

**要求**：
- 格式：PNG、JPG、WEBP 均可（推荐 PNG）
- 尺寸：建议 200x200 或 256x256
- 大小：小于 100KB

**示例放置：**
```
httpserver/
├── static/
│   ├── default_avatar.png  ← 放这里
│   └── README.md
```

### 2. 其他静态资源（可选）

如果需要添加其他静态资源，可以放在此目录：

```
httpserver/
├── static/
│   ├── default_avatar.png
│   ├── favicon.ico
│   ├── logo.png
│   └── css/
│       └── style.css
```

## 如何获取默认头像

### 方式1：生成占位图（推荐）

使用在线工具生成：
- https://placeholder.com/
- https://via.placeholder.com/200x200.png

下载后重命名为 `default_avatar.png`，放入此目录。

### 方式2：使用系统命令生成

**macOS/Linux（ImageMagick）：**
```bash
# 安装 ImageMagick
brew install imagemagick  # macOS
# 或 apt install imagemagick  # Linux

# 生成 200x200 灰色圆形头像
convert -size 200x200 xc:lightgray \
  -fill gray -draw "circle 100,100 100,10" \
  default_avatar.png
```

**Python（Pillow）：**
```python
from PIL import Image, ImageDraw

# 创建 200x200 灰色背景
img = Image.new('RGB', (200, 200), color='lightgray')
draw = ImageDraw.Draw(img)

# 绘制圆形
draw.ellipse([50, 50, 150, 150], fill='gray')

# 保存
img.save('default_avatar.png')
```

### 方式3：下载开源头像

- **Avatar Placeholder**: https://ui-avatars.com/
- **Gravatar Default**: https://www.gravatar.com/avatar/default
- **Open Peeps**: https://www.openpeeps.com/

## API 调用说明

### 获取默认头像

当用户未登录或未上传头像时，以下 API 会返回此默认头像：

```http
GET /api/v1/profile/picture
Authorization: (空或无效 Token)
```

### 响应

```
Content-Type: image/png
Content-Length: <文件大小>

<default_avatar.png 的二进制内容>
```

## 注意事项

⚠️ **重要提示**：

1. **文件必须存在**：如果 `default_avatar.png` 不存在，API 会返回 404 错误
2. **文件权限**：确保文件可被 HTTP Server 进程读取
3. **安全性**：不要在此目录放置敏感文件（如配置文件、密钥等）

## 验证安装

启动 HTTP Server 后，访问以下 URL 验证：

```bash
# 直接访问（不带 Token）
curl http://localhost:8080/api/v1/profile/picture -o test_avatar.png

# 检查文件
file test_avatar.png
# 输出应为：test_avatar.png: PNG image data, 200 x 200, ...
```

## 目录结构

完整的静态资源目录结构：

```
httpserver/
├── static/
│   ├── default_avatar.png  ← 默认头像（必需）
│   └── README.md           ← 此文件
├── uploads/
│   └── avatars/            ← 用户上传的头像存放在这里
│       ├── 123456_avatar.png
│       └── 789012_avatar.jpg
```

## 更新默认头像

如需更换默认头像：

1. 准备新的头像文件
2. 替换 `httpserver/static/default_avatar.png`
3. 重启 HTTP Server（或不需要重启，取决于实现）

**无需修改代码！**


